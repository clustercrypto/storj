// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <div class="token">
        <div class="token__icon">
            <div class="token__icon__wrapper">
                <StorjLarge />
            </div>
        </div>

        <div class="token__content">
            <div v-if="isLoading" class="token__content__loader-container">
                <v-loader />
            </div>
            <div v-else-if="!wallet.address" class="token__content__add-funds">
                <h3 class="token__content__add-funds__title">
                    STORJ Token
                </h3>
                <p class="token__content__add-funds__info">Deposit STORJ Token to your account and receive a 10% bonus, or $10 for every $100.</p>
                <!-- Claim wallet button -->
                <VButton
                    label="Add STORJ Tokens"
                    width="140px"
                    height="32px"
                    font-size="13px"
                    border-radius="6px"
                    :on-press="claimWalletClick"
                />
            </div>
            <template v-else>
                <div class="token__content__title-area">
                    <div class="token__content__title-area__small-icon">
                        <StorjSmall />
                    </div>
                    <div class="token__content__title-area__default-wrapper">
                        <p class="token__content__title-area__default-wrapper__label">Default</p>
                        <VInfo>
                            <template #icon>
                                <InfoIcon />
                            </template>
                            <template #message>
                                <p class="token__content__title-area__default-wrapper__message">
                                    If the STORJ token balance runs out, the default credit card will be charged.
                                    <a
                                        class="token__content__title-area__default-wrapper__message__link"
                                        href=""
                                        target="_blank"
                                        rel="noopener noreferrer"
                                    >
                                        Learn More
                                    </a>
                                </p>
                            </template>
                        </VInfo>
                    </div>
                </div>
                <div class="token__content__info-area">
                    <div class="token__content__info-area__option">
                        <h2 class="token__content__info-area__option__title">STORJ Token Deposit Address</h2>
                        <p class="token__content__info-area__option__value">{{ wallet.address }}</p>
                    </div>
                    <div class="token__content__info-area__option">
                        <h2 class="token__content__info-area__option__title">Total Balance</h2>
                        <p class="token__content__info-area__option__value">{{ wallet.balance.value }}</p>
                    </div>
                </div>
                <div class="token__content__action-area">
                    <VButton
                        class="token__content__action-area__history-btn"
                        label="See transactions"
                        :is-transparent="true"
                        height="32px"
                        font-size="13px"
                        border-radius="6px"
                        :on-press="() => $emit('showTransactions')"
                    />

                    <v-button
                        label="Add funds"
                        width="96px"
                        height="32px"
                        font-size="13px"
                        border-radius="6px"
                        :on-press="onAddTokensClick"
                    />
                </div>
            </template>
        </div>
    </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

import { APP_STATE_MUTATIONS } from '@/store/mutationConstants';
import { Wallet } from '@/types/payments';
import { AnalyticsHttpApi } from '@/api/analytics';
import { AnalyticsErrorEventSource, AnalyticsEvent } from '@/utils/constants/analyticsEventNames';
import { PAYMENTS_ACTIONS } from '@/store/modules/payments';

import VButton from '@/components/common/VButton.vue';
import VLoader from '@/components/common/VLoader.vue';
import VInfo from '@/components/common/VInfo.vue';

import InfoIcon from '@/../static/images/billing/blueInfoIcon.svg';
import StorjSmall from '@/../static/images/billing/storj-icon-small.svg';
import StorjLarge from '@/../static/images/billing/storj-icon-large.svg';

// @vue/component
@Component({
    components: {
        InfoIcon,
        StorjSmall,
        StorjLarge,
        VButton,
        VLoader,
        VInfo,
    },
})
export default class AddTokenCardNative extends Vue {
    private readonly analytics: AnalyticsHttpApi = new AnalyticsHttpApi();

    public isLoading = false;

    async mounted(): Promise<void> {
        await this.getWallet();

        // check if user navigated here from Billing overview screen
        if (this.$route.params.action !== 'add tokens') {
            return;
        }
        // user clicked 'Add Funds' on Billing overview screen.
        if (this.wallet.address) {
            this.onAddTokensClick();
        } else {
            await this.claimWalletClick();
        }
    }

    /**
     * getWallet tries to get an existing wallet for this user. this will not claim a wallet.
     */
    private async getWallet() {
        if (this.wallet.address) {
            return;
        }
        this.isLoading = true;
        await this.$store.dispatch(PAYMENTS_ACTIONS.GET_WALLET).catch(_ => {});
        this.isLoading = false;
    }

    /**
     * claimWallet claims a wallet for the current account.
     */
    private async claimWallet(): Promise<void> {
        if (!this.wallet.address)
            await this.$store.dispatch(PAYMENTS_ACTIONS.CLAIM_WALLET);
    }

    /**
     * Called when "Add STORJ Tokens" button is clicked.
     */
    public async claimWalletClick(): Promise<void> {
        this.isLoading = true;
        try {
            await this.claimWallet();
            // wallet claimed; open token modal
            this.onAddTokensClick();
        } catch (error) {
            await this.$notify.error(error.message, AnalyticsErrorEventSource.BILLING_STORJ_TOKEN_CONTAINER);
        }
        this.isLoading = false;
    }

    /**
     * Holds on add tokens button click logic.
     * Triggers Add funds popup.
     */
    public onAddTokensClick(): void {
        this.analytics.eventTriggered(AnalyticsEvent.ADD_FUNDS_CLICKED);
        this.$store.commit(APP_STATE_MUTATIONS.TOGGLE_ADD_TOKEN_FUNDS_MODAL_SHOWN);
    }

    /**
     * Returns wallet from store.
     */
    private get wallet(): Wallet {
        return this.$store.state.paymentsModule.wallet;
    }
}
</script>

<style scoped lang="scss">
    .token {
        border-radius: 10px;
        width: 300px;
        margin-right: 10px;
        padding: 24px;
        box-shadow: 0 0 20px rgb(0 0 0 / 4%);
        position: relative;
        font-family: 'font_regular', sans-serif;
        background-color: #fff;

        &__icon {
            position: absolute;
            top: 0;
            right: 0;
            width: 120px;
            height: 120px;
            overflow: hidden;

            &__wrapper {
                position: absolute;
                top: -20px;
                right: -20px;
            }
        }

        &__content {
            position: relative;
            height: 100%;

            &__loader-container {
                display: flex;
                align-items: center;
                justify-content: center;
                height: 100%;
                width: 100%;

                :deep(.loader) {
                    padding: 0;
                }
            }

            &__title {
                font-family: 'font_bold', sans-serif;
                font-size: 18px;
                line-height: 27px;
                color: #000;
                margin-bottom: 5px;
            }

            &__info {
                position: relative;
                font-size: 14px;
                line-height: 20px;
                color: var(--c-grey-6);
                margin-bottom: 23px;
                max-width: 232px;
            }

            &__title-area {
                position: relative;
                display: flex;
                align-items: center;
                justify-content: space-between;

                &__small-icon {
                    background: var(--c-blue-1);
                    border-radius: 4px;
                    width: 32px;
                    height: 24px;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                }

                &__default-wrapper {
                    display: flex;
                    align-items: center;
                    padding: 7px 8px;
                    background: var(--c-blue-1);
                    border: 1px solid #fff;
                    border-radius: 4px;

                    &__label {
                        font-family: 'font_bold', sans-serif;
                        font-size: 12px;
                        color: var(--c-blue-4);
                        margin-right: 4px;
                    }

                    &__message {
                        font-size: 12px;
                        line-height: 18px;
                        text-align: center;
                        color: #fff;
                        transform: rotate(180deg);

                        &__link {
                            color: #fff;
                            text-decoration: underline !important;
                            text-underline-position: under;
                        }
                    }

                    svg {
                        cursor: pointer;
                    }
                }
            }

            &__add-funds {
                display: flex;
                flex-direction: column;
                justify-content: space-between;
                height: 100%;
                width: 100%;

                &__title {
                    font-family: 'font_bold', sans-serif;
                }

                &__info {
                    font-size: 14px;
                    line-height: 20px;
                    color: #000;

                    a {
                        color: var(--c-blue-3);
                        text-decoration: underline !important;
                    }
                }
            }

            &__info-area {
                position: relative;
                display: flex;
                align-items: center;
                margin: 32px 0 25px;

                &__option {

                    &:first-of-type {
                        margin-right: 16px;
                        max-width: 185px;
                    }

                    &__title {
                        font-family: 'font_medium', sans-serif;
                        font-size: 12px;
                        line-height: 18px;
                        color: var(--c-grey-6);
                    }

                    &__value {
                        font-family: 'font_bold', sans-serif;
                        font-size: 16px;
                        line-height: 24px;
                        color: #000;

                        &:first-of-type {
                            white-space: nowrap;
                            text-overflow: ellipsis;
                            overflow: hidden;
                        }
                    }
                }
            }

            &__action-area {
                display: flex;
                justify-content: start;
                align-items: center;
                gap: 10px;

                &__history-btn {
                    cursor: pointer;
                    padding: 0 10px;

                    span {
                        font-size: 13px;
                        color: var(--c-grey-6);
                        font-family: 'font_medium', sans-serif;
                        line-height: 23px;
                        margin: 0;
                        white-space: nowrap;
                    }
                }
            }
        }
    }

    :deep(.info__box) {
        transform: rotate(-180deg);
        top: calc(100% - 100px);
        left: calc(100% - 123px);
        filter: none;
    }

    :deep(.info__box__message) {
        padding: 8px 8px 13px;
        width: 235px;
        background: var(--c-grey-6);
        border-radius: 4px;
    }

    :deep(.info__box__arrow) {
        width: 10px;
        height: 10px;
        background: var(--c-grey-6);
        margin-bottom: -3px;
    }
</style>
