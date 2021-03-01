// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package metainfo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"
	"golang.org/x/time/rate"

	"storj.io/common/uuid"
	"storj.io/storj/satellite/metainfo/metabase"
)

const batchsizeLimit = 2500

var (
	// LoopError is a standard error class for this component.
	LoopError = errs.Class("metainfo loop error")
	// LoopClosedError is a loop closed error.
	LoopClosedError = LoopError.New("loop closed")
)

// Object is the object info passed to Observer by metainfo loop.
type Object metabase.LoopObjectEntry

// Expired checks if object expired relative to now.
func (object *Object) Expired(now time.Time) bool {
	return object.ExpiresAt != nil && object.ExpiresAt.Before(now)
}

// Segment is the segment info passed to Observer by metainfo loop.
type Segment struct {
	Location       metabase.SegmentLocation // tally, repair, graceful exit, audit
	CreationDate   time.Time                // repair
	ExpirationDate time.Time                // tally, repair
	LastRepaired   time.Time                // repair

	metabase.LoopSegmentEntry
}

// Expired checks if segment is expired relative to now.
func (segment *Segment) Expired(now time.Time) bool {
	return !segment.ExpirationDate.IsZero() && segment.ExpirationDate.Before(now)
}

// Observer is an interface defining an observer that can subscribe to the metainfo loop.
//
// architecture: Observer
type Observer interface {
	Object(context.Context, *Object) error
	RemoteSegment(context.Context, *Segment) error
	InlineSegment(context.Context, *Segment) error
}

// NullObserver is an observer that does nothing. This is useful for joining
// and ensuring the metainfo loop runs once before you use a real observer.
type NullObserver struct{}

// Object implements the Observer interface.
func (NullObserver) Object(context.Context, *Object) error {
	return nil
}

// RemoteSegment implements the Observer interface.
func (NullObserver) RemoteSegment(context.Context, *Segment) error {
	return nil
}

// InlineSegment implements the Observer interface.
func (NullObserver) InlineSegment(context.Context, *Segment) error {
	return nil
}

type observerContext struct {
	observer Observer

	ctx  context.Context
	done chan error

	object *monkit.DurationDist
	remote *monkit.DurationDist
	inline *monkit.DurationDist
}

func newObserverContext(ctx context.Context, obs Observer) *observerContext {
	name := fmt.Sprintf("%T", obs)
	key := monkit.NewSeriesKey("observer").WithTag("name", name)

	return &observerContext{
		observer: obs,

		ctx:  ctx,
		done: make(chan error),

		object: monkit.NewDurationDist(key.WithTag("pointer_type", "object")),
		inline: monkit.NewDurationDist(key.WithTag("pointer_type", "inline")),
		remote: monkit.NewDurationDist(key.WithTag("pointer_type", "remote")),
	}
}

func (observer *observerContext) Object(ctx context.Context, object *Object) error {
	start := time.Now()
	defer func() { observer.object.Insert(time.Since(start)) }()

	return observer.observer.Object(ctx, object)
}

func (observer *observerContext) RemoteSegment(ctx context.Context, segment *Segment) error {
	start := time.Now()
	defer func() { observer.remote.Insert(time.Since(start)) }()

	return observer.observer.RemoteSegment(ctx, segment)
}

func (observer *observerContext) InlineSegment(ctx context.Context, segment *Segment) error {
	start := time.Now()
	defer func() { observer.inline.Insert(time.Since(start)) }()

	return observer.observer.InlineSegment(ctx, segment)
}

func (observer *observerContext) HandleError(err error) bool {
	if err != nil {
		observer.done <- err
		observer.Finish()
		return true
	}
	return false
}

func (observer *observerContext) Finish() {
	close(observer.done)

	name := fmt.Sprintf("%T", observer.observer)
	stats := allObserverStatsCollectors.GetStats(name)
	stats.Observe(observer)
}

func (observer *observerContext) Wait() error {
	return <-observer.done
}

// LoopConfig contains configurable values for the metainfo loop.
type LoopConfig struct {
	CoalesceDuration time.Duration `help:"how long to wait for new observers before starting iteration" releaseDefault:"5s" devDefault:"5s"`
	RateLimit        float64       `help:"rate limit (default is 0 which is unlimited segments per second)" default:"0"`
	ListLimit        int           `help:"how many items to query in a batch" default:"2500"`
}

// Loop is a metainfo loop service.
//
// architecture: Service
type Loop struct {
	config     LoopConfig
	metabaseDB MetabaseDB
	join       chan []*observerContext
	done       chan struct{}
}

// NewLoop creates a new metainfo loop service.
func NewLoop(config LoopConfig, metabaseDB MetabaseDB) *Loop {
	return &Loop{
		metabaseDB: metabaseDB,
		config:     config,
		join:       make(chan []*observerContext),
		done:       make(chan struct{}),
	}
}

// Join will join the looper for one full cycle until completion and then returns.
// On ctx cancel the observer will return without completely finishing.
// Only on full complete iteration it will return nil.
// Safe to be called concurrently.
func (loop *Loop) Join(ctx context.Context, observers ...Observer) (err error) {
	defer mon.Task()(&ctx)(&err)

	obsContexts := make([]*observerContext, len(observers))
	for i, obs := range observers {
		obsContexts[i] = newObserverContext(ctx, obs)
	}

	select {
	case loop.join <- obsContexts:
	case <-ctx.Done():
		return ctx.Err()
	case <-loop.done:
		return LoopClosedError
	}

	var errList errs.Group
	for _, ctx := range obsContexts {
		errList.Add(ctx.Wait())
	}

	return errList.Err()
}

// Run starts the looping service.
// It can only be called once, otherwise a panic will occur.
func (loop *Loop) Run(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)

	for {
		err := loop.RunOnce(ctx)
		if err != nil {
			return err
		}
	}
}

// Close closes the looping services.
func (loop *Loop) Close() (err error) {
	close(loop.done)
	return nil
}

// RunOnce goes through metainfo one time and sends information to observers.
//
// It is not safe to call this concurrently with Run.
func (loop *Loop) RunOnce(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)

	var observers []*observerContext

	// wait for the first observer, or exit because context is canceled
	select {
	case list := <-loop.join:
		observers = append(observers, list...)
	case <-ctx.Done():
		return ctx.Err()
	}

	// after the first observer is found, set timer for CoalesceDuration and add any observers that try to join before the timer is up
	timer := time.NewTimer(loop.config.CoalesceDuration)
waitformore:
	for {
		select {
		case list := <-loop.join:
			observers = append(observers, list...)
		case <-timer.C:
			break waitformore
		case <-ctx.Done():
			finishObservers(observers)
			return ctx.Err()
		}
	}
	return iterateDatabase(ctx, loop.metabaseDB, observers, loop.config.ListLimit, rate.NewLimiter(rate.Limit(loop.config.RateLimit), 1))
}

// IterateDatabase iterates over PointerDB and notifies specified observers about results.
//
// It uses 10000 as the lookup limit for iterating.
func IterateDatabase(ctx context.Context, rateLimit float64, metabaseDB MetabaseDB, observers ...Observer) error {
	obsContexts := make([]*observerContext, len(observers))
	for i, observer := range observers {
		obsContexts[i] = newObserverContext(ctx, observer)
	}
	return iterateDatabase(ctx, metabaseDB, obsContexts, 10000, rate.NewLimiter(rate.Limit(rateLimit), 1))
}

// Wait waits for run to be finished.
// Safe to be called concurrently.
func (loop *Loop) Wait() {
	<-loop.done
}

func iterateDatabase(ctx context.Context, metabaseDB MetabaseDB, observers []*observerContext, limit int, rateLimiter *rate.Limiter) (err error) {
	defer func() {
		if err != nil {
			for _, observer := range observers {
				observer.HandleError(err)
			}
			return
		}
		finishObservers(observers)
	}()

	observers, err = iterateObjects(ctx, metabaseDB, observers, limit, rateLimiter)
	if err != nil {
		return LoopError.Wrap(err)
	}

	return err
}

func iterateObjects(ctx context.Context, metabaseDB MetabaseDB, observers []*observerContext, limit int, rateLimiter *rate.Limiter) (_ []*observerContext, err error) {
	defer mon.Task()(&ctx)(&err)

	if limit <= 0 || limit > batchsizeLimit {
		limit = batchsizeLimit
	}

	startingTime := time.Now()

	noObserversErr := errs.New("no observers")

	// TODO we may consider keeping only expiration time as its
	// only thing we need to handle segments
	objectsMap := make(map[uuid.UUID]metabase.LoopObjectEntry)
	ids := make([]uuid.UUID, 0, limit)

	processBatch := func() error {
		if len(objectsMap) == 0 {
			return nil
		}

		err := metabaseDB.IterateLoopStreams(ctx, metabase.IterateLoopStreams{
			StreamIDs:      ids,
			AsOfSystemTime: startingTime,
		}, func(ctx context.Context, streamID uuid.UUID, next metabase.SegmentIterator) error {
			if err := ctx.Err(); err != nil {
				return err
			}

			obj, ok := objectsMap[streamID]
			if !ok {
				return Error.New("unable to find corresponding object: %v", streamID)
			}
			delete(objectsMap, streamID)

			observers = withObservers(observers, func(observer *observerContext) bool {
				object := Object(obj)
				return handleObject(ctx, observer, &object)
			})
			if len(observers) == 0 {
				return noObserversErr
			}

			for {
				// if context has been canceled exit. Otherwise, continue
				if err := ctx.Err(); err != nil {
					return err
				}

				var segment metabase.LoopSegmentEntry
				if !next(&segment) {
					break
				}

				location := metabase.SegmentLocation{
					ProjectID:  obj.ProjectID,
					BucketName: obj.BucketName,
					ObjectKey:  obj.ObjectKey,
					Position:   segment.Position,
				}

				observers = withObservers(observers, func(observer *observerContext) bool {
					return handleSegment(ctx, observer, location, segment, obj.ExpiresAt)
				})
				if len(observers) == 0 {
					return noObserversErr
				}
			}

			return nil
		})
		if err != nil {
			return Error.Wrap(err)
		}

		if len(objectsMap) > 0 {
			return Error.New("unhandled objects %#v", objectsMap)
		}

		return nil
	}

	segmentsInBatch := int32(0)
	err = metabaseDB.IterateLoopObjects(ctx, metabase.IterateLoopObjects{
		BatchSize: limit,
	}, func(ctx context.Context, it metabase.LoopObjectsIterator) error {
		var entry metabase.LoopObjectEntry
		for it.Next(ctx, &entry) {
			if err := rateLimiter.Wait(ctx); err != nil {
				// We don't really execute concurrent batches so we should never
				// exceed the burst size of 1 and this should never happen.
				// We can also enter here if the context is cancelled.
				return err
			}

			objectsMap[entry.StreamID] = entry
			ids = append(ids, entry.StreamID)

			// add +1 to reduce risk of crossing limit
			segmentsInBatch += entry.SegmentCount + 1

			if segmentsInBatch >= int32(limit) {
				err := processBatch()
				if err != nil {
					if errors.Is(err, noObserversErr) {
						return nil
					}
					return err
				}

				if len(objectsMap) > 0 {
					return errs.New("objects map is not empty")
				}

				ids = ids[:0]
				segmentsInBatch = 0
			}
		}
		err = processBatch()
		if errors.Is(err, noObserversErr) {
			return nil
		}
		return err
	})

	return observers, err
}

func withObservers(observers []*observerContext, handleObserver func(observer *observerContext) bool) []*observerContext {
	nextObservers := observers[:0]
	for _, observer := range observers {
		keepObserver := handleObserver(observer)
		if keepObserver {
			nextObservers = append(nextObservers, observer)
		}
	}
	return nextObservers
}

func handleObject(ctx context.Context, observer *observerContext, object *Object) bool {
	if observer.HandleError(observer.Object(ctx, object)) {
		return false
	}

	select {
	case <-observer.ctx.Done():
		observer.HandleError(observer.ctx.Err())
		return false
	default:
	}

	return true
}

func handleSegment(ctx context.Context, observer *observerContext, location metabase.SegmentLocation, segment metabase.LoopSegmentEntry, expirationDate *time.Time) bool {
	loopSegment := &Segment{
		Location: location,
		// TODO we are not setting this since multipart-upload branch, we need to
		// check if thats affecting anything and if we need to set it correctly
		// or just replace it with something else
		CreationDate: time.Time{},
		// TODO we are not setting this and we need to decide what to do with this
		LastRepaired: time.Time{},

		LoopSegmentEntry: segment,
	}

	if expirationDate != nil {
		loopSegment.ExpirationDate = *expirationDate
	}

	if loopSegment.Inline() {
		if observer.HandleError(observer.InlineSegment(ctx, loopSegment)) {
			return false
		}
	} else {
		if observer.HandleError(observer.RemoteSegment(ctx, loopSegment)) {
			return false
		}
	}

	select {
	case <-observer.ctx.Done():
		observer.HandleError(observer.ctx.Err())
		return false
	default:
	}

	return true
}

func finishObservers(observers []*observerContext) {
	for _, observer := range observers {
		observer.Finish()
	}
}
