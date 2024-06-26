// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <div class="encrypt">
        <h2 class="encrypt__title">Select Encryption</h2>
        <div
            v-if="!(encryptSelect === 'create' && (isPassphraseDownloaded || isPassphraseCopied))"
            class="encrypt__item"
        >
            <div class="encrypt__item__left-area">
                <AccessKeyIcon
                    class="encrypt__item__left-area__icon"
                    :class="{ selected: encryptSelect === 'generate' }"
                />
                <div class="encrypt__item__left-area__text">
                    <h3>Generate Passphrase</h3>
                    <p>Automatically Generate Seed</p>
                </div>
            </div>
            <div class="encrypt__item__radio">
                <input
                    id="generate-check"
                    v-model="encryptSelect"
                    value="generate"
                    type="radio"
                    name="type"
                    @change="onRadioInput"
                >
            </div>
        </div>
        <div
            v-if="encryptSelect === 'generate'"
            class="encrypt__generated-passphrase"
        >
            {{ passphrase }}
        </div>
        <div
            v-if="!(encryptSelect && (isPassphraseDownloaded || isPassphraseCopied))"
            id="divider"
            class="encrypt__divider"
            :class="{ 'in-middle': encryptSelect === 'generate' }"
        />
        <div
            v-if="!(encryptSelect === 'generate' && (isPassphraseDownloaded || isPassphraseCopied))"
            id="own"
            :class="{ 'in-middle': encryptSelect === 'generate' }"
            class="encrypt__item"
        >
            <div class="encrypt__item__left-area">
                <ThumbPrintIcon
                    class="encrypt__item__left-area__icon"
                    :class="{ selected: encryptSelect === 'create' }"
                />
                <div class="encrypt__item__left-area__text">
                    <h3>Create My Own Passphrase</h3>
                    <p>Make it Personalized</p>
                </div>
            </div>
            <div class="encrypt__item__radio">
                <input
                    id="create-check"
                    v-model="encryptSelect"
                    value="create"
                    type="radio"
                    name="type"
                    @change="onRadioInput"
                >
            </div>
        </div>
        <input
            v-if="encryptSelect === 'create'"
            v-model="passphrase"
            type="text"
            placeholder="Input Your Passphrase"
            class="encrypt__passphrase" :disabled="encryptSelect === 'generate'"
            @input="resetSavedStatus"
        >
        <div
            class="encrypt__footer-container"
            :class="{ 'in-middle': encryptSelect === 'generate' }"
        >
            <div class="encrypt__footer-container__buttons">
                <v-button
                    v-clipboard:copy="passphrase"
                    :label="isPassphraseCopied ? 'Copied' : 'Copy to clipboard'"
                    height="50px"
                    :is-transparent="!isPassphraseCopied"
                    :is-white-green="isPassphraseCopied"
                    class="encrypt__footer-container__buttons__copy-button"
                    font-size="14px"
                    :on-press="onCopyPassphraseClick"
                    :is-disabled="passphrase.length < 1"
                >
                    <template v-if="!isPassphraseCopied" #icon>
                        <copy-icon class="button-icon" :class="{ active: passphrase }" />
                    </template>
                </v-button>
                <v-button
                    label="Download .txt"
                    font-size="14px"
                    height="50px"
                    class="encrypt__footer-container__buttons__download-button"
                    :is-green-white="isPassphraseDownloaded"
                    :on-press="downloadPassphrase"
                    :is-disabled="passphrase.length < 1"
                >
                    <template v-if="!isPassphraseDownloaded" #icon>
                        <download-icon class="button-icon" />
                    </template>
                </v-button>
            </div>
            <div v-if="isPassphraseDownloaded || isPassphraseCopied" :class="`encrypt__footer-container__acknowledgement-container ${acknowledgementCheck ? 'blue-background' : ''}`">
                <input
                    id="acknowledgement"
                    v-model="acknowledgementCheck"
                    type="checkbox"
                    class="encrypt__footer-container__acknowledgement-container__check"
                >
                <label for="acknowledgement" class="encrypt__footer-container__acknowledgement-container__text">I understand that Storj does not know or store my encryption passphrase. If I lose it, I won't be able to recover files.</label>
            </div>
            <div
                v-if="isPassphraseDownloaded || isPassphraseCopied"
                class="encrypt__footer-container__buttons"
            >
                <v-button
                    label="Back"
                    height="50px"
                    :is-transparent="true"
                    class="encrypt__footer-container__buttons__copy-button"
                    font-size="14px"
                    :on-press="backAction"
                />
                <v-button
                    label="Create my Access ⟶"
                    font-size="14px"
                    height="50px"
                    class="encrypt__footer-container__buttons__download-button"
                    :is-disabled="!acknowledgementCheck"
                    :on-press="createAccessGrant"
                />
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { generateMnemonic } from 'bip39';
import { ref, watch } from 'vue';

import { Download } from '@/utils/download';
import { AnalyticsHttpApi } from '@/api/analytics';
import { AnalyticsEvent } from '@/utils/constants/analyticsEventNames';
import { useNotify } from '@/utils/hooks';

import VButton from '@/components/common/VButton.vue';

import CopyIcon from '@/../static/images/common/copy.svg';
import DownloadIcon from '@/../static/images/common/download.svg';
import AccessKeyIcon from '@/../static/images/accessGrants/accessKeyIcon.svg';
import ThumbPrintIcon from '@/../static/images/accessGrants/thumbPrintIcon.svg';

const notify = useNotify();

const emit = defineEmits(['apply-passphrase', 'create-access', 'close-modal', 'backAction']);

const encryptSelect = ref<'create' | 'generate'>('create');
const isPassphraseCopied = ref<boolean>(false);
const isPassphraseDownloaded = ref<boolean>(false);
const acknowledgementCheck = ref<boolean>(false);
const passphrase = ref<string>('');

const currentDate = new Date().toISOString();
const analytics: AnalyticsHttpApi = new AnalyticsHttpApi();

function createAccessGrant(): void {
    emit('create-access');
}

function onCloseClick(): void {
    emit('close-modal');
}

function onRadioInput(): void {
    isPassphraseCopied.value = false;
    isPassphraseDownloaded.value = false;
    passphrase.value = '';

    if (encryptSelect.value === 'generate') {
        passphrase.value = generateMnemonic();
    }
}

function backAction(): void {
    emit('backAction');
}

function resetSavedStatus(): void {
    isPassphraseCopied.value = false;
    isPassphraseDownloaded.value = false;
}

function onCopyPassphraseClick(): void {
    isPassphraseCopied.value = true;
    analytics.eventTriggered(AnalyticsEvent.COPY_TO_CLIPBOARD_CLICKED);
    notify.success(`Passphrase was copied successfully`);
}

/**
 * Downloads passphrase to .txt file
 */
function downloadPassphrase(): void {
    isPassphraseDownloaded.value = true;
    Download.file(passphrase.value, `passphrase-${currentDate}.txt`);
    analytics.eventTriggered(AnalyticsEvent.DOWNLOAD_TXT_CLICKED);
}

watch(passphrase, (newPassphrase) => {
    emit('apply-passphrase', newPassphrase);
});
</script>

<style scoped lang="scss">
.button-icon {
    margin-right: 5px;

    :deep(path),
    :deep(rect) {
        stroke: white;
    }

    &.active {

        :deep(path),
        :deep(rect) {
            stroke: var(--c-grey-6);
        }
    }
}

.encrypt {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    justify-content: center;
    font-family: 'font_regular', sans-serif;
    padding: 32px;
    max-width: 350px;

    &__title {
        font-family: 'font_bold', sans-serif;
        font-size: 28px;
        line-height: 36px;
        letter-spacing: -0.02em;
        color: #000;
        margin-bottom: 32px;
    }

    &__divider {
        width: 100%;
        height: 1px;
        background: var(--c-grey-2);
        margin: 16px 0;

        &.in-middle {
            order: 4;
        }
    }

    &__item {
        display: flex;
        align-items: center;
        justify-content: space-between;
        width: 100%;
        box-sizing: border-box;
        margin-top: 10px;

        &__left-area {
            display: flex;
            align-items: center;
            justify-content: flex-start;

            &__icon {
                margin-right: 8px;

                &.selected {

                    :deep(circle) {
                        fill: var(--c-blue-1) !important;
                    }

                    :deep(path) {
                        fill: var(--c-blue-4) !important;
                    }
                }
            }

            &__text {
                display: flex;
                flex-direction: column;
                justify-content: space-between;
                align-items: flex-start;
                font-family: 'font_regular', sans-serif;
                font-size: 12px;

                h3 {
                    margin: 0 0 8px;
                    font-family: 'font_bold', sans-serif;
                    font-size: 14px;
                }

                p {
                    padding: 0;
                }
            }
        }

        &__radio {
            display: flex;
            align-items: center;
            justify-content: center;
            width: 10px;
            height: 10px;
        }
    }

    &__generated-passphrase {
        margin-top: 20px;
        margin-bottom: 20px;
        align-items: center;
        padding: 10px 16px;
        background: var(--c-grey-2);
        border: 1px solid var(--c-grey-4);
        border-radius: 7px;
        text-align: left;
    }

    &__passphrase {
        margin-top: 20px;
        width: 100%;
        background: #fff;
        border: 1px solid var(--c-grey-4);
        box-sizing: border-box;
        border-radius: 4px;
        font-size: 14px;
        padding: 10px;
    }

    &__footer-container {
        display: flex;
        flex-direction: column;
        width: 100%;
        justify-content: flex-start;
        margin-top: 16px;

        &__buttons {
            display: flex;
            width: 100%;
            margin-top: 25px;
            column-gap: 8px;

            @media screen and (max-width: 390px) {
                flex-direction: column;
                column-gap: unset;
                row-gap: 8px;
            }

            &__copy-button,
            &__download-button {
                padding: 0 15px;

                @media screen and (max-width: 390px) {
                    width: unset !important;
                }
            }
        }

        &__acknowledgement-container {
            border: 1px solid var(--c-grey-4);
            border-radius: 6px;
            display: grid;
            grid-template-columns: 1fr 6fr;
            padding: 10px;
            margin-top: 25px;
            height: 80px;
            align-content: center;

            &__check {
                margin: 0 auto auto;
                border-radius: 4px;
                height: 16px;
                width: 16px;
            }

            &__text {
                text-align: left;
            }
        }
    }
}
</style>
