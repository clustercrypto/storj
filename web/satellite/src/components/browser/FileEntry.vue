// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <table-item
        v-if="fileTypeIsFile"
        :selected="isFileSelected"
        :on-click="selectFile"
        :on-primary-click="openModal"
        :item="{'name': file.Key, 'size': size, 'date': uploadDate}"
        table-type="file"
    >
        <th slot="options" v-click-outside="closeDropdown" class="file-entry__functional options overflow-visible" @click.stop="openDropdown">
            <div
                v-if="loadingSpinner()"
                class="spinner-border"
                role="status"
            />
            <dots-icon v-else />
            <div v-if="dropdownOpen" class="file-entry__functional__dropdown">
                <div class="file-entry__functional__dropdown__item" @click.stop="openModal">
                    <details-icon />
                    <p class="file-entry__functional__dropdown__item__label">Details</p>
                </div>

                <div class="file-entry__functional__dropdown__item" @click.stop="download">
                    <download-icon />
                    <p class="file-entry__functional__dropdown__item__label">Download</p>
                </div>

                <div class="file-entry__functional__dropdown__item" @click.stop="share">
                    <share-icon />
                    <p class="file-entry__functional__dropdown__item__label">Share</p>
                </div>

                <div v-if="!deleteConfirmation" class="file-entry__functional__dropdown__item" @click.stop="confirmDeletion">
                    <delete-icon />
                    <p class="file-entry__functional__dropdown__item__label">Delete</p>
                </div>
                <div v-else class="file-entry__functional__dropdown__item confirmation">
                    <div class="delete-confirmation">
                        <p class="delete-confirmation__text">
                            Are you sure?
                        </p>
                        <div class="delete-confirmation__options">
                            <span class="delete-confirmation__options__item yes" @click.stop="finalDelete">
                                <span><delete-icon /></span>
                                <span>Yes</span>
                            </span>

                            <span class="delete-confirmation__options__item no" @click.stop="cancelDeletion">
                                <span><close-icon /></span>
                                <span>No</span>
                            </span>
                        </div>
                    </div>
                </div>
            </div>
        </th>
    </table-item>
    <table-item
        v-else-if="fileTypeIsFolder"
        :item="{'name': file.Key, 'size': '', 'date': ''}"
        :selected="isFileSelected"
        :on-click="selectFile"
        :on-primary-click="openBucket"
        table-type="folder"
    >
        <th slot="options" v-click-outside="closeDropdown" class="file-entry__functional options overflow-visible" @click.stop="openDropdown">
            <div
                v-if="loadingSpinner()"
                class="spinner-border"
                role="status"
            />
            <dots-icon v-else />
            <div v-if="dropdownOpen" class="file-entry__functional__dropdown">
                <div
                    v-if="!deleteConfirmation" class="file-entry__functional__dropdown__item"
                    @click.stop="confirmDeletion"
                >
                    <delete-icon />
                    <p class="file-entry__functional__dropdown__item__label">Delete</p>
                </div>
                <div v-else class="file-entry__functional__dropdown__item confirmation">
                    <div class="delete-confirmation">
                        <p class="delete-confirmation__text">
                            Are you sure?
                        </p>
                        <div class="delete-confirmation__options">
                            <span class="delete-confirmation__options__item yes" @click.stop="finalDelete">
                                <span><delete-icon /></span>
                                <span>Yes</span>
                            </span>

                            <span class="delete-confirmation__options__item no" @click.stop="cancelDeletion">
                                <span><close-icon /></span>
                                <span>No</span>
                            </span>
                        </div>
                    </div>
                </div>
            </div>
        </th>
    </table-item>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import prettyBytes from 'pretty-bytes';

import type { BrowserFile } from '@/types/browser';
import { APP_STATE_MUTATIONS } from '@/store/mutationConstants';
import { useNotify, useRouter, useStore } from '@/utils/hooks';
import { AnalyticsErrorEventSource } from '@/utils/constants/analyticsEventNames';

import TableItem from '@/components/common/TableItem.vue';

import DeleteIcon from '@/../static/images/objects/delete.svg';
import ShareIcon from '@/../static/images/objects/share.svg';
import DetailsIcon from '@/../static/images/objects/details.svg';
import DownloadIcon from '@/../static/images/objects/download.svg';
import DotsIcon from '@/../static/images/objects/dots.svg';
import CloseIcon from '@/../static/images/common/closeCross.svg';

const store = useStore();
const notify = useNotify();
const router = useRouter();

const props = defineProps<{
  path: string,
  file: BrowserFile,
}>();

const emit = defineEmits(['onUpdate']);

const deleteConfirmation = ref(false);

/**
 * Return the size of the file formatted.
 */
const size = computed((): string => {
    return prettyBytes(props.file.Size);
});

/**
 * Return the upload date of the file formatted.
 */
const uploadDate = computed((): string => {
    return props.file.LastModified.toLocaleString().split(',')[0];
});

/**
 * Check with the store to see if the dropdown is open for the current file/folder.
 */
const dropdownOpen = computed((): boolean => {
    return store.state.files.openedDropdown === props.file.Key;
});

/**
 * Return a link to the current folder for navigation.
 */
const link = computed((): string => {
    const browserRoot = store.state.files.browserRoot;
    const uriParts = (store.state.files.path + props.file.Key).split('/');
    const pathAndKey = uriParts.map(part => encodeURIComponent(part)).join('/');
    return pathAndKey.length > 0
        ? browserRoot + pathAndKey + '/'
        : browserRoot;
});

/**
 * Return a flag signifying whether the current file/folder is selected.
 */
const isFileSelected = computed((): boolean => {
    return Boolean(
        store.state.files.selectedAnchorFile === props.file ||
      store.state.files.selectedFiles.find(
          (file) => file === props.file,
      ) ||
      store.state.files.shiftSelectedFiles.find(
          (file) => file === props.file,
      ),
    );
});

/**
 * Return a boolean signifying whether the current file/folder is a folder.
 */
const fileTypeIsFolder = computed((): boolean => {
    return props.file.type === 'folder';
});

/**
 * Return a boolean signifying whether the current file/folder is a file.
 */
const fileTypeIsFile = computed((): boolean => {
    return props.file.type === 'file';
});

/**
 * Open the modal for the current file.
 */
function openModal(): void {
    store.commit('files/setObjectPathForModal', props.path + props.file.Key);
    store.commit(APP_STATE_MUTATIONS.TOGGLE_OBJECT_DETAILS_MODAL_SHOWN);
    store.dispatch('files/closeDropdown');
}

/**
 * Return a boolean signifying whether the current file/folder is in the process of being deleted, therefore a spinner shoud be shown.
 */
function loadingSpinner(): boolean {
    return Boolean(store.state.files.filesToBeDeleted.find(
        (file) => file === props.file,
    ));
}

/**
 * Select the current file/folder whether it be a click, click + shiftKey, click + metaKey or ctrlKey, or unselect the rest.
 */
function selectFile(event: KeyboardEvent): void {
    if (store.state.files.openedDropdown) {
        store.dispatch('files/closeDropdown');
    }

    if (event.shiftKey) {
        setShiftSelectedFiles();

        return;
    }

    const isSelectedFile = Boolean(event.metaKey || event.ctrlKey);

    setSelectedFile(isSelectedFile);
}

async function openBucket(): Promise<void> {
    await router.push(link.value);
    emit('onUpdate');
}

/**
 * Set the selected file/folder in the store.
 */
function setSelectedFile(command: boolean): void {
    /* this function is responsible for selecting and unselecting a file on file click or [CMD + click] AKA command. */
    const shiftSelectedFiles =
      store.state.files.shiftSelectedFiles;
    const selectedFiles = store.state.files.selectedFiles;

    const files = [
        ...selectedFiles,
        ...shiftSelectedFiles,
    ];

    const selectedAnchorFile =
      store.state.files.selectedAnchorFile;

    if (command && props.file === selectedAnchorFile) {
    /* if it's [CMD + click] and the file selected is the actual selectedAnchorFile, then unselect the file but store it under unselectedAnchorFile in case the user decides to do a [shift + click] right after this action. */

        store.commit('files/setUnselectedAnchorFile', props.file);
        store.commit('files/setSelectedAnchorFile', null);
    } else if (command && files.includes(props.file)) {
    /* if it's [CMD + click] and the file selected is a file that has already been selected in selectedFiles and shiftSelectedFiles, then unselect it by filtering it out. */

        store.dispatch(
            'files/updateSelectedFiles',
            selectedFiles.filter(
                (fileSelected) => fileSelected !== props.file,
            ),
        );

        store.dispatch(
            'files/updateShiftSelectedFiles',
            shiftSelectedFiles.filter(
                (fileSelected) => fileSelected !== props.file,
            ),
        );
    } else if (command && selectedAnchorFile) {
    /* if it's [CMD + click] and there is already a selectedAnchorFile, then add the selectedAnchorFile and shiftSelectedFiles into the array of selectedFiles, set selectedAnchorFile to the file that was clicked, set unselectedAnchorFile to null, and set shiftSelectedFiles to an empty array. */

        const filesSelected = [...selectedFiles];

        if (!filesSelected.includes(selectedAnchorFile)) {
            filesSelected.push(selectedAnchorFile);
        }

        store.dispatch('files/updateSelectedFiles', [
            ...filesSelected,
            ...shiftSelectedFiles.filter(
                (file) => !filesSelected.includes(file),
            ),
        ]);

        store.commit('files/setSelectedAnchorFile', props.file);
        store.commit('files/setUnselectedAnchorFile', null);
        store.dispatch('files/updateShiftSelectedFiles', []);
    } else if (command) {
    /* if it's [CMD + click] and it has not met any of the above conditions, then set selectedAnchorFile to file and set unselectedAnchorfile to null, update the selectedFiles, and update the shiftSelectedFiles */

        store.commit('files/setSelectedAnchorFile', props.file);
        store.commit('files/setUnselectedAnchorFile', null);

        store.dispatch('files/updateSelectedFiles', [
            ...selectedFiles,
            ...shiftSelectedFiles,
        ]);

        store.dispatch('files/updateShiftSelectedFiles', []);
    } else {
    /* if it's just a file click without any modifier, then set selectedAnchorFile to the file that was clicked, set shiftSelectedFiles and selectedFiles to an empty array. */

        store.commit('files/setSelectedAnchorFile', props.file);
        store.dispatch('files/updateShiftSelectedFiles', []);
        store.dispatch('files/updateSelectedFiles', []);
    }
}

/**
 * Set files/folders selected using shift key in the store.
 */
function setShiftSelectedFiles(): void {
    /* this function is responsible for selecting all files from selectedAnchorFile to the file that was selected with [shift + click] */

    const files = store.getters['files/sortedFiles'];
    const unselectedAnchorFile =
      store.state.files.unselectedAnchorFile;

    if (unselectedAnchorFile) {
    /* if there is an unselectedAnchorFile, meaning that in the previous action the user unselected the anchor file but is now chosing to do a [shift + click] on another file, then reset the selectedAnchorFile, the achor file, to unselectedAnchorFile. */
        store.commit(
            'files/setSelectedAnchorFile',
            unselectedAnchorFile,
        );

        store.commit('files/setUnselectedAnchorFile', null);
    }

    const selectedAnchorFile = store.state.files.selectedAnchorFile;

    if (!selectedAnchorFile) {
        store.commit('files/setSelectedAnchorFile', props.file);

        return;
    }

    const anchorIdx = files.findIndex(
        (file) => file === selectedAnchorFile,
    );
    const shiftIdx = files.findIndex((file) => file === props.file);

    const start = Math.min(anchorIdx, shiftIdx);
    const end = Math.max(anchorIdx, shiftIdx) + 1;

    store.dispatch(
        'files/updateShiftSelectedFiles',
        files
            .slice(start, end)
            .filter(
                (file) =>
                    !store.state.files.selectedFiles.includes(
                        file,
                    ) && file !== selectedAnchorFile,
            ),
    );
}

/**
 * Open the share modal for the current file.
 */
function share(): void {
    store.dispatch('files/closeDropdown');
    store.commit('files/setObjectPathForModal', props.path + props.file.Key);
    store.commit(APP_STATE_MUTATIONS.TOGGLE_SHARE_OBJECT_MODAL_SHOWN);
}

/**
 * Close the dropdown.
 */
function closeDropdown(): void {
    store.dispatch('files/closeDropdown');

    // remove the dropdown delete confirmation
    deleteConfirmation.value = false;
}

/**
 * Open the dropdown for the current file/folder.
 */
function openDropdown(): void {
    store.dispatch('files/openDropdown', props.file.Key);

    // remove the dropdown delete confirmation
    deleteConfirmation.value = false;
}

/**
 * Download the current file.
 */
function download(): void {
    try {
        store.dispatch('files/download', props.file);
        notify.warning('Do not share download link with other people. If you want to share this data better use "Share" option.');
    } catch (error) {
        notify.error('Can not download your file', AnalyticsErrorEventSource.FILE_BROWSER_ENTRY);
    }

    store.dispatch('files/closeDropdown');
    deleteConfirmation.value = false;
}

/**
 * Set the data property deleteConfirmation to true, signifying that this user does in fact want the current selected file/folder.
 */
function confirmDeletion(): void {
    deleteConfirmation.value = true;
}

/**
 * Delete the selected file/folder.
 */
async function finalDelete(): Promise<void> {
    store.dispatch('files/closeDropdown');
    store.dispatch('files/addFileToBeDeleted', props.file);

    const params = { ...props };

    (props.file.type === 'file') ? await store.dispatch('files/delete', params) : store.dispatch('files/deleteFolder', params);

    // refresh the files displayed
    try {
        await store.dispatch('files/list');
    } catch (error) {
        notify.error(error.message, AnalyticsErrorEventSource.FILE_BROWSER_ENTRY);
    }

    store.dispatch('files/removeFileFromToBeDeleted', props.file);
    deleteConfirmation.value = false;
}

/**
 * Abort the deletion of the current file/folder.
 */
function cancelDeletion(): void {
    store.dispatch('files/closeDropdown');
    deleteConfirmation.value = false;
}
</script>

<style scoped lang="scss">
.file-entry {

    &__functional {
        padding: 0 10px;
        position: relative;
        cursor: pointer;

        &__dropdown {
            position: absolute;
            top: 25px;
            right: 15px;
            background: #fff;
            box-shadow: 0 20px 34px rgb(10 27 44 / 28%);
            border-radius: 6px;
            width: 255px;
            z-index: 100;

            &__item {
                display: flex;
                align-items: center;
                padding: 20px 25px;
                width: calc(100% - 50px);

                .dropdown-item.action.p-3.action {
                    font-family: 'Inter', sans-serif;
                }

                &__label {
                    margin: 0 0 0 10px;
                }

                &:not(.confirmation):hover {
                    background-color: #f4f5f7;
                    font-family: 'font_medium', sans-serif;
                    color: var(--c-blue-3);

                    svg :deep(path) {
                        fill: var(--c-blue-3);
                    }
                }
            }
        }
    }
}

.delete-confirmation {
    display: flex;
    flex-direction: column;
    gap: 5px;
    align-items: start;
    width: 100%;

    &__options {
        display: flex;
        gap: 20px;
        align-items: center;

        &__item {
            display: flex;
            gap: 5px;
            align-items: center;

            &.yes:hover {
                color: var(--c-red-2);

                svg :deep(path) {
                    fill: var(--c-red-2);
                    stroke: var(--c-red-2);
                }
            }

            &.no:hover {
                color: var(--c-blue-3);

                svg :deep(path) {
                    fill: var(--c-blue-3);
                    stroke: var(--c-blue-3);
                }
            }
        }
    }
}

@media screen and (max-width: 550px) {
    // hide size, upload date columns on mobile screens

    :deep(.data:not(:first-of-type)) {
        display: none;
    }
}

@keyframes spinner-border {

    to {
        transform: rotate(360deg);
    }
}

.spinner-border {
    display: inline-block;
    width: 2rem;
    height: 2rem;
    vertical-align: text-bottom;
    border: 0.25em solid currentcolor;
    border-right-color: transparent;
    border-radius: 50%;
    animation: 0.75s linear infinite spinner-border;
}
</style>
