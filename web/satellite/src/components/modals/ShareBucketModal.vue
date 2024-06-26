// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <VModal :on-close="closeModal">
        <template #content>
            <div class="modal">
                <h1 class="modal__title">Share Bucket</h1>
                <ShareContainer :link="link" />
                <p class="modal__label">
                    Or copy link:
                </p>
                <VLoader v-if="isLoading" width="20px" height="20px" />
                <div v-if="!isLoading" class="modal__input-group">
                    <input
                        id="url"
                        class="modal__input"
                        type="url"
                        :value="link"
                        aria-describedby="btn-copy-link"
                        readonly
                    >
                    <VButton
                        :label="copyButtonState === ButtonStates.Copy ? 'Copy' : 'Copied'"
                        width="114px"
                        height="40px"
                        :on-press="onCopy"
                        :is-disabled="isLoading"
                        :is-green-white="copyButtonState === ButtonStates.Copied"
                        :icon="copyButtonState === ButtonStates.Copied ? 'none' : 'copy'"
                    />
                </div>
            </div>
        </template>
    </VModal>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

import { APP_STATE_MUTATIONS } from '@/store/mutationConstants';
import { ACCESS_GRANTS_ACTIONS } from '@/store/modules/accessGrants';
import { PROJECTS_ACTIONS } from '@/store/modules/projects';
import { MetaUtils } from '@/utils/meta';
import { AccessGrant, EdgeCredentials } from '@/types/accessGrants';
import { AnalyticsErrorEventSource } from '@/utils/constants/analyticsEventNames';

import VModal from '@/components/common/VModal.vue';
import VLoader from '@/components/common/VLoader.vue';
import VButton from '@/components/common/VButton.vue';
import ShareContainer from '@/components/common/share/ShareContainer.vue';

enum ButtonStates {
    Copy,
    Copied,
}

// @vue/component
@Component({
    components: {
        VModal,
        VButton,
        VLoader,
        ShareContainer,
    },
})
export default class ShareBucketModal extends Vue {
    private worker: Worker;
    private readonly ButtonStates = ButtonStates;

    public isLoading = true;
    public link = '';
    public copyButtonState = ButtonStates.Copy;

    /**
     * Lifecycle hook after initial render.
     * Sets local worker.
     */
    public async mounted(): Promise<void> {
        this.setWorker();
        await this.setShareLink();
    }

    /**
     * Copies link to users clipboard.
     */
    public async onCopy(): Promise<void> {
        await this.$copyText(this.link);
        this.copyButtonState = ButtonStates.Copied;

        setTimeout(() => {
            this.copyButtonState = ButtonStates.Copy;
        }, 2000);

        await this.$notify.success('Link copied successfully.');
    }

    /**
     * Sets share bucket link.
     */
    private async setShareLink(): Promise<void> {
        try {
            let path = `${this.bucketName}`;
            const now = new Date();
            const LINK_SHARING_AG_NAME = `${path}_shared-bucket_${now.toISOString()}`;
            const cleanAPIKey: AccessGrant = await this.$store.dispatch(ACCESS_GRANTS_ACTIONS.CREATE, LINK_SHARING_AG_NAME);

            const satelliteNodeURL = MetaUtils.getMetaContent('satellite-nodeurl');
            const salt = await this.$store.dispatch(PROJECTS_ACTIONS.GET_SALT, this.$store.getters.selectedProject.id);

            this.worker.postMessage({
                'type': 'GenerateAccess',
                'apiKey': cleanAPIKey.secret,
                'passphrase': this.passphrase,
                'salt': salt,
                'satelliteNodeURL': satelliteNodeURL,
            });

            const grantEvent: MessageEvent = await new Promise(resolve => this.worker.onmessage = resolve);
            const grantData = grantEvent.data;
            if (grantData.error) {
                await this.$notify.error(grantData.error, AnalyticsErrorEventSource.SHARE_BUCKET_MODAL);

                return;
            }

            this.worker.postMessage({
                'type': 'RestrictGrant',
                'isDownload': true,
                'isUpload': false,
                'isList': true,
                'isDelete': false,
                'paths': [path],
                'grant': grantData.value,
            });

            const event: MessageEvent = await new Promise(resolve => this.worker.onmessage = resolve);
            const data = event.data;
            if (data.error) {
                await this.$notify.error(data.error, AnalyticsErrorEventSource.SHARE_BUCKET_MODAL);

                return;
            }

            const credentials: EdgeCredentials =
                await this.$store.dispatch(ACCESS_GRANTS_ACTIONS.GET_GATEWAY_CREDENTIALS, { accessGrant: data.value, isPublic: true });

            path = encodeURIComponent(path.trim());

            const linksharingURL = MetaUtils.getMetaContent('linksharing-url');

            this.link = `${linksharingURL}/${credentials.accessKeyId}/${path}`;
        } catch (error) {
            await this.$notify.error(error.message, AnalyticsErrorEventSource.SHARE_BUCKET_MODAL);
        } finally {
            this.isLoading = false;
        }
    }

    /**
     * Sets local worker with worker instantiated in store.
     */
    public setWorker(): void {
        this.worker = this.$store.state.accessGrantsModule.accessGrantsWebWorker;
        this.worker.onerror = (error: ErrorEvent) => {
            this.$notify.error(error.message, AnalyticsErrorEventSource.SHARE_BUCKET_MODAL);
        };
    }

    /**
     * Closes open bucket modal.
     */
    public closeModal(): void {
        if (this.isLoading) return;

        this.$store.commit(APP_STATE_MUTATIONS.TOGGLE_SHARE_BUCKET_MODAL_SHOWN);
    }

    /**
     * Returns chosen bucket name from store.
     */
    private get bucketName(): string {
        return this.$store.state.objectsModule.fileComponentBucketName;
    }

    /**
     * Returns passphrase from store.
     */
    private get passphrase(): string {
        return this.$store.state.objectsModule.passphrase;
    }
}
</script>

<style scoped lang="scss">
    .modal {
        font-family: 'font_regular', sans-serif;
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 50px;
        max-width: 470px;

        @media screen and (max-width: 430px) {
            padding: 20px;
        }

        &__title {
            font-family: 'font_bold', sans-serif;
            font-size: 22px;
            line-height: 29px;
            color: #1b2533;
            margin: 10px 0 35px;
        }

        &__label {
            font-family: 'font_medium', sans-serif;
            font-size: 14px;
            line-height: 21px;
            color: #354049;
            align-self: center;
            margin: 20px 0 10px;
        }

        &__link {
            font-size: 16px;
            line-height: 21px;
            color: #384b65;
            align-self: flex-start;
            word-break: break-all;
            text-align: left;
        }

        &__buttons {
            display: flex;
            column-gap: 20px;
            margin-top: 32px;
            width: 100%;

            @media screen and (max-width: 430px) {
                flex-direction: column-reverse;
                column-gap: unset;
                row-gap: 15px;
            }
        }

        &__input-group {
            border: 1px solid var(--c-grey-4);
            background: var(--c-grey-1);
            padding: 10px;
            display: flex;
            justify-content: space-between;
            border-radius: 8px;
            width: 100%;
            height: 42px;
        }

        &__input {
            background: none;
            border: none;
            font-size: 14px;
            color: var(--c-grey-6);
            outline: none;
            max-width: 340px;
            width: 100%;

            @media screen and (max-width: 430px) {
                max-width: 210px;
            }
        }
    }
</style>
