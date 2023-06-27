package vesting_test

import (
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
)

type HandlerTestSuite struct {
	suite.Suite

	handler sdk.Handler
	app     *simapp.SimApp
}

func (suite *HandlerTestSuite) SetupTest() {
	checkTx := false
	app := simapp.Setup(suite.T(), checkTx)

	suite.handler = vesting.NewHandler(
		app.AccountKeeper,
		app.BankKeeper,
		app.DistrKeeper,
		app.StakingKeeper,
	)
	suite.app = app
}

func (suite *HandlerTestSuite) TestMsgCreateVestingAccount() {
	ctx := suite.app.BaseApp.NewContext(false, tmproto.Header{Height: suite.app.LastBlockHeight() + 1})

	balances := sdk.NewCoins(sdk.NewInt64Coin("test", 1000))
	addr1 := sdk.AccAddress([]byte("addr1_______________"))
	addr2 := sdk.AccAddress([]byte("addr2_______________"))
	addr3 := sdk.AccAddress([]byte("addr3_______________"))

	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(ctx, addr1)
	suite.app.AccountKeeper.SetAccount(ctx, acc1)
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, ctx, addr1, balances))

	testCases := []struct {
		name      string
		msg       *types.MsgCreateVestingAccount
		expectErr bool
	}{
		{
			name:      "create delayed vesting account",
			msg:       types.NewMsgCreateVestingAccount(addr1, addr2, sdk.NewCoins(sdk.NewInt64Coin("test", 100)), ctx.BlockTime().Unix()+10000, true),
			expectErr: false,
		},
		{
			name:      "create continuous vesting account",
			msg:       types.NewMsgCreateVestingAccount(addr1, addr3, sdk.NewCoins(sdk.NewInt64Coin("test", 100)), ctx.BlockTime().Unix()+10000, false),
			expectErr: false,
		},
		{
			name:      "continuous vesting account already exists",
			msg:       types.NewMsgCreateVestingAccount(addr1, addr3, sdk.NewCoins(sdk.NewInt64Coin("test", 100)), ctx.BlockTime().Unix()+10000, false),
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			res, err := suite.handler(ctx, tc.msg)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				toAddr, err := sdk.AccAddressFromBech32(tc.msg.ToAddress)
				suite.Require().NoError(err)
				accI := suite.app.AccountKeeper.GetAccount(ctx, toAddr)
				suite.Require().NotNil(accI)

				if tc.msg.Delayed {
					acc, ok := accI.(*types.DelayedVestingAccount)
					suite.Require().True(ok)
					suite.Require().Equal(tc.msg.Amount, acc.GetVestingCoins(ctx.BlockTime()))
				} else {
					acc, ok := accI.(*types.ContinuousVestingAccount)
					suite.Require().True(ok)
					suite.Require().Equal(tc.msg.Amount, acc.GetVestingCoins(ctx.BlockTime()))
				}
			}
		})
	}
}

func (suite *HandlerTestSuite) TestMsgDonateVestingToken() {
	ctx := suite.app.BaseApp.NewContext(false, tmproto.Header{Height: suite.app.LastBlockHeight() + 1})

	prevCommunityFund := suite.app.DistrKeeper.GetFeePool(ctx).CommunityPool

	balances := sdk.NewCoins(sdk.NewInt64Coin("test", 1000))
	addr1 := sdk.AccAddress([]byte("addr1_______________"))
	addr2 := sdk.AccAddress([]byte("addr2_______________"))
	addr3 := sdk.AccAddress([]byte("addr3_______________"))
	addr4 := sdk.AccAddress([]byte("addr4_______________"))

	valAddr := sdk.ValAddress([]byte("validator___________"))
	suite.app.StakingKeeper.SetValidator(ctx, stakingtypes.Validator{
		OperatorAddress:   valAddr.String(),
		ConsensusPubkey:   nil,
		Jailed:            false,
		Status:            0,
		Tokens:            sdk.NewInt(2),
		DelegatorShares:   sdk.MustNewDecFromStr("1.1"),
		Description:       stakingtypes.Description{},
		UnbondingHeight:   0,
		UnbondingTime:     time.Time{},
		Commission:        stakingtypes.Commission{},
		MinSelfDelegation: sdk.NewInt(1),
	})

	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(ctx, addr1)
	suite.app.AccountKeeper.SetAccount(ctx, acc1)
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, ctx, addr1, balances))

	acc2 := types.NewPermanentLockedAccount(
		suite.app.AccountKeeper.NewAccountWithAddress(ctx, addr2).(*authtypes.BaseAccount), balances,
	)
	acc2.DelegatedVesting = balances
	suite.app.AccountKeeper.SetAccount(ctx, acc2)
	suite.app.StakingKeeper.SetDelegation(ctx, stakingtypes.Delegation{
		DelegatorAddress: addr2.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           sdk.OneDec(),
	})
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, ctx, addr2, balances))

	acc3 := types.NewPermanentLockedAccount(
		suite.app.AccountKeeper.NewAccountWithAddress(ctx, addr3).(*authtypes.BaseAccount), balances,
	)
	suite.app.AccountKeeper.SetAccount(ctx, acc3)
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, ctx, addr3, balances))

	acc4 := types.NewPermanentLockedAccount(
		suite.app.AccountKeeper.NewAccountWithAddress(ctx, addr4).(*authtypes.BaseAccount), balances,
	)
	acc4.DelegatedVesting = balances
	suite.app.AccountKeeper.SetAccount(ctx, acc4)
	suite.app.StakingKeeper.SetDelegation(ctx, stakingtypes.Delegation{
		DelegatorAddress: addr4.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           sdk.MustNewDecFromStr("0.1"),
	})
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, ctx, addr4, balances))

	testCases := []struct {
		name      string
		msg       *types.MsgDonateAllVestingTokens
		expectErr bool
	}{
		{
			name:      "donate from normal account",
			msg:       types.NewMsgDonateAllVestingTokens(addr1),
			expectErr: true,
		},
		{
			name:      "donate from vesting account with delegated vesting",
			msg:       types.NewMsgDonateAllVestingTokens(addr2),
			expectErr: true,
		},
		{
			name:      "donate from vesting account",
			msg:       types.NewMsgDonateAllVestingTokens(addr3),
			expectErr: false,
		},
		{
			name:      "donate from vesting account with dust delegation",
			msg:       types.NewMsgDonateAllVestingTokens(addr4),
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// Rollback context after every test case
			ctx, _ := ctx.CacheContext()
			res, err := suite.handler(ctx, tc.msg)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				feePool := suite.app.DistrKeeper.GetFeePool(ctx).CommunityPool.Sub(prevCommunityFund)
				communityFund, _ := feePool.TruncateDecimal()
				suite.Require().Equal(balances, communityFund)

				fromAddr, err := sdk.AccAddressFromBech32(tc.msg.FromAddress)
				suite.Require().NoError(err)
				accI := suite.app.AccountKeeper.GetAccount(ctx, fromAddr)
				suite.Require().NotNil(accI)
				_, ok := accI.(*authtypes.BaseAccount)
				suite.Require().True(ok)
				balance := suite.app.BankKeeper.GetAllBalances(ctx, fromAddr)
				suite.Require().Empty(balance)

				_, broken := stakingkeeper.DelegatorSharesInvariant(suite.app.StakingKeeper)(ctx)
				suite.Require().False(broken)
			}
		})
	}
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
