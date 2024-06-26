// Copyright (C) 2023 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <div class="dashboard-header">
        <VLoader
            v-if="loading"
            class="dashboard-header__loader"
            width="100px"
            height="100px"
        />
        <template v-else>
            <template v-if="promptForPassphrase && !bucketWasCreated">
                <p class="dashboard-header__subtitle">
                    Set an encryption passphrase <br>to start uploading files.
                </p>
                <VButton
                    label="Set Encryption Passphrase ->"
                    width="234px"
                    height="48px"
                    font-size="14px"
                    :on-press="onSetClick"
                />
            </template>
            <template v-else-if="!promptForPassphrase && !bucketWasCreated && !bucketsPage.buckets.length && !bucketsPage.search">
                <p class="dashboard-header__subtitle">
                    Create a bucket to start <br>uploading data in your project.
                </p>
                <VButton
                    label="Create a bucket ->"
                    width="160px"
                    height="48px"
                    font-size="14px"
                    :on-press="onCreateBucketClick"
                />
            </template>
            <template v-else>
                <p class="dashboard-header__subtitle" aria-roledescription="with-usage-title">
                    Your
                    <span class="dashboard-header__subtitle__value">{{ limits.objectCount }} objects</span>
                    are stored <br>in
                    <span class="dashboard-header__subtitle__value">{{ limits.segmentCount }} segments</span>
                    around the world
                </p>
                <p class="dashboard-header__limits">
                    <span class="dashboard-header__limits--bold">Storage Limit</span>
                    per month: {{ limits.storageLimit | bytesToBase10String }} |
                    <span class="dashboard-header__limits--bold">Bandwidth Limit</span>
                    per month: {{ limits.bandwidthLimit | bytesToBase10String }}
                </p>
                <VButton
                    label="Upload"
                    width="100px"
                    height="40px"
                    :on-press="onUploadClick"
                />
            </template>
        </template>
    </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { useRouter, useStore } from '@/utils/hooks';
import { APP_STATE_MUTATIONS } from '@/store/mutationConstants';
import { BucketPage } from '@/types/buckets';
import { ProjectLimits } from '@/types/projects';
import { RouteConfig } from '@/router';
import { LocalData } from '@/utils/localData';

import VButton from '@/components/common/VButton.vue';
import VLoader from '@/components/common/VLoader.vue';

const props = withDefaults(defineProps<{
    loading?: boolean;
}>(), {
    loading: false,
});

const store = useStore();
const router = useRouter();

/**
 * Indicates if user should be prompt for passphrase.
 */
const promptForPassphrase = computed((): boolean => {
    return store.state.objectsModule.promptForPassphrase;
});

/**
 * Indicates if bucket was created.
 */
const bucketWasCreated = computed((): boolean => {
    const status = LocalData.getBucketWasCreatedStatus();
    if (status !== null) {
        return status;
    }

    return false;
});

/**
 * Returns current limits from store.
 */
const limits = computed((): ProjectLimits => {
    return store.state.projectsModule.currentLimits;
});

/**
 * Returns fetched buckets page from store.
 */
const bucketsPage = computed((): BucketPage => {
    return store.state.bucketUsageModule.page;
});

/**
 * Toggles create project passphrase modal visibility.
 */
function onSetClick() {
    store.commit(APP_STATE_MUTATIONS.TOGGLE_CREATE_PROJECT_PASSPHRASE_MODAL_SHOWN);
}

/**
 * Toggles create bucket modal visibility.
 */
function onCreateBucketClick() {
    store.commit(APP_STATE_MUTATIONS.TOGGLE_CREATE_BUCKET_MODAL_SHOWN);
}

/**
 * Redirects to bucket management screen.
 */
function onUploadClick() {
    router.push(RouteConfig.Buckets.with(RouteConfig.BucketsManagement).path);
}
</script>

<style scoped lang="scss">
.dashboard-header {
    font-family: 'font_regular', sans-serif;

    &__loader {
        display: inline-block;
    }

    &__subtitle {
        font-family: 'font_bold', sans-serif;
        font-size: 28px;
        line-height: 36px;
        letter-spacing: -0.02em;
        color: #000;
        margin-bottom: 16px;

        &__value {
            text-decoration: underline;
            text-underline-position: under;
            text-decoration-color: var(--c-green-3);
        }
    }

    &__limits {
        font-size: 14px;
        margin: 11px 0 16px;

        &--bold {
            font-family: 'font_bold', sans-serif;
        }
    }
}
</style>
