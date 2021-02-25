package ui

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/decred/dcrd/dcrutil"
	"github.com/planetdecred/dcrlibwallet"
	"github.com/planetdecred/godcr/ui/decredmaterial"
	"github.com/planetdecred/godcr/ui/values"
	"github.com/planetdecred/godcr/wallet"
)

const (
	PageSend               = "Send"
	invalidPassphraseError = "error broadcasting transaction: " + dcrlibwallet.ErrInvalidPassphrase
)

type httpError interface {
	Error() string
}

type amountValue struct {
	inactiveInputAmount      string
	leftTransactionFeeValue  string
	rightTransactionFeeValue string
	leftTotalCostValue       string
	rightTotalCostValue      string
}

type sendPage struct {
	pageContainer, walletsList,
	accountsList layout.List
	theme *decredmaterial.Theme

	fromAccountBtn *widget.Clickable

	txAuthor        *dcrlibwallet.TxAuthor
	broadcastResult *wallet.Broadcast

	wallet          *wallet.Wallet
	selectedWallet  wallet.InfoShort
	selectedAccount **wallet.Account

	toAddress    *widget.Bool
	isMoreOption bool

	unspentOutputsSelected *map[int]map[int32]map[string]*wallet.UnspentOutput

	destinationAddressEditor decredmaterial.Editor
	leftAmountEditor         decredmaterial.Editor
	rightAmountEditor        decredmaterial.Editor
	passwordEditor           decredmaterial.Editor

	currencySwap, moreOption decredmaterial.IconButton

	// customChangeAddressEditor    decredmaterial.Editor
	sendAmountEditor             decredmaterial.Editor
	nextButton                   decredmaterial.Button
	closeConfirmationModalButton decredmaterial.Button
	confirmButton                decredmaterial.Button
	maxButton                    decredmaterial.Button
	sendToButton                 decredmaterial.Button
	clearAllBtn                  decredmaterial.Button

	accountSwitch *decredmaterial.SwitchButtonText

	confirmModal       *decredmaterial.Modal
	walletAccountModal *decredmaterial.Modal

	txFeeCollapsible *decredmaterial.Collapsible

	remainingBalance int64
	amountAtoms      int64
	totalCostDCR     int64
	txFee            int64
	spendableBalance int64

	usdExchangeRate float64
	inputAmount     float64
	amountUSDtoDCR  float64
	amountDCRtoUSD  float64

	count int

	amountErrorText    string
	calculateErrorText string

	activeTotalAmount   string
	inactiveInputAmount string
	currencyValue       string

	leftExchangeValue  string
	rightExchangeValue string

	leftTransactionFeeValue  string
	rightTransactionFeeValue string

	leftTotalCostValue  string
	rightTotalCostValue string

	// walletName     string
	// accountName    string
	// accountBalance string

	balanceAfterSendValue string

	LastTradeRate string
	exchangeErr   string

	passwordModal *decredmaterial.Password
	line          *decredmaterial.Line

	isConfirmationModalOpen   bool
	isPasswordModalOpen       bool
	isBroadcastingTransaction bool

	isWalletAccountModalOpen bool

	shouldInitializeTxAuthor bool

	txAuthorErrChan chan error

	broadcastErrChan chan error

	errChan chan error

	toggleCoinCtrl      *widget.Bool
	inputButtonCoinCtrl decredmaterial.Button

	toAcctDetails []*gesture.Click
}

func (win *Window) SendPage(common pageCommon) layout.Widget {
	pg := &sendPage{
		pageContainer: layout.List{
			Axis:      layout.Vertical,
			Alignment: layout.Middle,
		},
		accountsList: layout.List{
			Axis: layout.Vertical,
		},

		walletsList: layout.List{
			Axis: layout.Vertical,
		},

		theme:                  common.theme,
		wallet:                 common.wallet,
		txAuthor:               &win.txAuthor,
		broadcastResult:        &win.broadcastResult,
		unspentOutputsSelected: &common.selectedUTXO,
		selectedAccount:        &win.walletAccount,
		errChan:                common.errorChannels[PageCreateRestore],

		fromAccountBtn: new(widget.Clickable),
		toAddress:      new(widget.Bool),

		accountSwitch:      common.theme.SwitchButtonText("Address", "My Account", new(widget.Clickable), new(widget.Clickable)),
		leftExchangeValue:  "DCR",
		rightExchangeValue: "USD",

		closeConfirmationModalButton: common.theme.Button(new(widget.Clickable), "Cancel"),
		nextButton:                   common.theme.Button(new(widget.Clickable), "Next"),
		confirmButton:                common.theme.Button(new(widget.Clickable), ""),
		maxButton:                    common.theme.Button(new(widget.Clickable), "MAX"),
		clearAllBtn:                  common.theme.Button(new(widget.Clickable), "Clear all fields"),
		txFeeCollapsible:             common.theme.Collapsible(),

		confirmModal:              common.theme.Modal(),
		walletAccountModal:        common.theme.Modal(),
		isConfirmationModalOpen:   false,
		isPasswordModalOpen:       false,
		isBroadcastingTransaction: false,
		isWalletAccountModalOpen:  false,

		passwordModal:    common.theme.Password(),
		broadcastErrChan: make(chan error),
		txAuthorErrChan:  make(chan error),
		line:             common.theme.Line(),
	}

	pg.toAcctDetails = make([]*gesture.Click, 0)

	pg.line.Color = common.theme.Color.Background
	pg.line.Height = 2

	pg.balanceAfterSendValue = "- DCR"

	activeEditorHint := fmt.Sprintf("Amount (%s)", pg.leftExchangeValue)
	pg.leftAmountEditor = common.theme.Editor(new(widget.Editor), activeEditorHint)
	pg.leftAmountEditor.Editor.SetText("")
	pg.leftAmountEditor.IsCustomButton = true
	pg.leftAmountEditor.Editor.SingleLine = true
	pg.leftAmountEditor.CustomButton.Background = common.theme.Color.Gray
	pg.leftAmountEditor.CustomButton.Inset = layout.UniformInset(values.MarginPadding2)
	pg.leftAmountEditor.CustomButton.Text = "Max"
	pg.leftAmountEditor.CustomButton.CornerRadius = values.MarginPadding0

	inactiveEditorHint := fmt.Sprintf("Amount (%s)", pg.rightExchangeValue)
	pg.rightAmountEditor = common.theme.Editor(new(widget.Editor), inactiveEditorHint)
	pg.rightAmountEditor.Editor.SetText("")
	pg.rightAmountEditor.IsCustomButton = true
	pg.rightAmountEditor.Editor.SingleLine = true
	pg.rightAmountEditor.CustomButton.Background = common.theme.Color.Gray
	pg.rightAmountEditor.CustomButton.Inset = layout.UniformInset(values.MarginPadding2)
	pg.rightAmountEditor.CustomButton.Text = "Max"
	pg.rightAmountEditor.CustomButton.CornerRadius = values.MarginPadding0

	editorColor := common.theme.Color.Primary
	pg.passwordEditor = common.theme.Editor(new(widget.Editor), "Spending password")
	pg.passwordEditor.IsTitleLabel = true
	pg.passwordEditor.Editor.SetText("")
	pg.passwordEditor.Editor.SingleLine = true
	pg.passwordEditor.Editor.Mask = '*'
	pg.passwordEditor.LineColor, pg.passwordEditor.TitleLabelColor = editorColor, editorColor

	pg.destinationAddressEditor = common.theme.Editor(new(widget.Editor), "Address")
	pg.destinationAddressEditor.Editor.SingleLine, pg.destinationAddressEditor.IsVisible = true, true
	pg.destinationAddressEditor.Editor.SetText("")

	// pg.customChangeAddressEditor = common.theme.Editor(new(widget.Editor), "Custom Change Address")
	// pg.customChangeAddressEditor.IsVisible, pg.customChangeAddressEditor.IsTitleLabel = true, true
	// pg.customChangeAddressEditor.Editor.SetText("")
	// pg.customChangeAddressEditor.Editor.SingleLine = true

	pg.closeConfirmationModalButton.Background = color.NRGBA{}
	pg.closeConfirmationModalButton.Color = common.theme.Color.Primary

	pg.currencySwap = common.theme.IconButton(new(widget.Clickable), common.icons.actionSwapHoriz)
	pg.currencySwap.Background = color.NRGBA{}
	pg.currencySwap.Color = common.theme.Color.Text
	pg.currencySwap.Inset = layout.UniformInset(values.MarginPadding0)
	pg.currencySwap.Size = values.MarginPadding25

	pg.moreOption = common.theme.IconButton(new(widget.Clickable), common.icons.navMoreIcon)
	pg.moreOption.Background = color.NRGBA{}
	pg.moreOption.Color = common.theme.Color.Text
	pg.moreOption.Inset = layout.UniformInset(values.MarginPadding0)

	pg.maxButton.Background = common.theme.Color.Black
	pg.maxButton.Inset = layout.UniformInset(values.MarginPadding5)

	pg.sendToButton = common.theme.Button(new(widget.Clickable), "Send to account")
	pg.sendToButton.TextSize = values.TextSize14
	pg.sendToButton.Background = color.NRGBA{}
	pg.sendToButton.Color = common.theme.Color.Primary
	pg.sendToButton.Inset = layout.UniformInset(values.MarginPadding0)

	pg.clearAllBtn.Background = common.theme.Color.Surface
	pg.clearAllBtn.Color = common.theme.Color.Text
	pg.clearAllBtn.Inset = layout.UniformInset(values.MarginPadding15)

	pg.toggleCoinCtrl = new(widget.Bool)
	pg.inputButtonCoinCtrl = common.theme.Button(new(widget.Clickable), "Inputs")
	pg.inputButtonCoinCtrl.Inset = layout.UniformInset(values.MarginPadding5)
	pg.inputButtonCoinCtrl.TextSize = values.MarginPadding10

	go common.wallet.GetUSDExchangeValues(&pg, pg.errChan)

	return func(gtx C) D {
		pg.Handle(common)
		return pg.Layout(gtx, common)
	}
}

func (pg *sendPage) Layout(gtx layout.Context, common pageCommon) layout.Dimensions {
	if *pg.selectedAccount == nil {
		pg.selectedWallet = common.info.Wallets[0]
		*pg.selectedAccount = &common.info.Wallets[0].Accounts[0]
		pg.shouldInitializeTxAuthor = true
	}

	pageContent := []func(gtx C) D{
		func(gtx C) D {
			return pg.topNav(gtx, common)
		},
		func(gtx C) D {
			return pg.fromSection(gtx, common)
		},
		func(gtx C) D {
			return pg.toSection(gtx, common)
		},
		func(gtx C) D {
			return pg.feeSection(gtx)
		},
	}

	dims := common.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Stack{Alignment: layout.NE}.Layout(gtx,
					layout.Expanded(func(gtx C) D {
						return common.UniformPadding(gtx, func(gtx C) D {
							return pg.pageContainer.Layout(gtx, len(pageContent), func(gtx C, i int) D {
								return layout.Inset{Bottom: values.MarginPadding10}.Layout(gtx, pageContent[i])
							})
						})
					}),
					layout.Stacked(func(gtx C) D {
						if pg.isMoreOption {
							inset := layout.Inset{
								Top:   values.MarginPadding40,
								Right: values.MarginPadding20,
							}
							return inset.Layout(gtx, func(gtx C) D {
								border := widget.Border{Color: pg.theme.Color.Background, CornerRadius: values.MarginPadding5, Width: values.MarginPadding1}
								return border.Layout(gtx, func(gtx C) D {
									return pg.clearAllBtn.Layout(gtx)
								})
							})
						}
						return layout.Dimensions{}
					}),
				)
			}),
			layout.Flexed(1, func(gtx C) D {
				return layout.S.Layout(gtx, func(gtx C) D {
					return layout.Inset{Left: values.MarginPadding1}.Layout(gtx, func(gtx C) D {
						return pg.balanceSection(gtx, common)
					})
				})
			}),
		)
	})

	if pg.isConfirmationModalOpen {
		return common.Modal(gtx, dims, pg.confirmationModal(gtx, common))
	}

	if pg.isWalletAccountModalOpen {
		return common.Modal(gtx, dims, pg.walletAccountSection(gtx, common))
	}

	return dims
}

func (pg *sendPage) topNav(gtx layout.Context, common pageCommon) layout.Dimensions {
	m := values.MarginPadding15
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					common.subPageBackButton.Icon = common.icons.contentClear
					return common.subPageBackButton.Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					txt := pg.theme.H6("Send DCR")
					txt.Color = pg.theme.Color.Text
					return layout.Inset{Left: m}.Layout(gtx, func(gtx C) D {
						return txt.Layout(gtx)
					})
				}),
			)
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.E.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return common.subPageInfoButton.Layout(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return layout.Inset{Left: m}.Layout(gtx, func(gtx C) D {
							return pg.moreOption.Layout(gtx)
						})
					}),
				)
			})
		}),
	)
}

func (pg *sendPage) fromSection(gtx layout.Context, common pageCommon) layout.Dimensions {
	return pg.pageSections(gtx, "From", func(gtx C) D {
		border := widget.Border{Color: pg.theme.Color.Background, CornerRadius: values.MarginPadding5, Width: values.MarginPadding2}
		return border.Layout(gtx, func(gtx C) D {
			return layout.UniformInset(values.MarginPadding15).Layout(gtx, func(gtx C) D {
				return decredmaterial.Clickable(gtx, pg.fromAccountBtn, func(gtx C) D {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							accountIcon := common.icons.accountIcon
							accountIcon.Scale = 0.9

							inset := layout.Inset{
								Right: values.MarginPadding10,
								Top:   values.MarginPadding2,
							}
							return inset.Layout(gtx, func(gtx C) D {
								return accountIcon.Layout(gtx)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return pg.theme.Body1((*pg.selectedAccount).Name).Layout(gtx)
						}),
						layout.Rigid(func(gtx C) D {
							inset := layout.Inset{
								Left: values.MarginPadding5,
								Top:  values.MarginPadding2,
							}
							return inset.Layout(gtx, func(gtx C) D {
								return decredmaterial.Card{
									Color: pg.theme.Color.Background,
								}.Layout(gtx, func(gtx C) D {
									return layout.UniformInset(values.MarginPadding2).Layout(gtx, func(gtx C) D {
										text := pg.theme.Caption(pg.selectedWallet.Name)
										text.Color = pg.theme.Color.Gray
										return text.Layout(gtx)
									})
								})
							})
						}),
						layout.Flexed(1, func(gtx C) D {
							return layout.E.Layout(gtx, func(gtx C) D {
								return layout.Flex{}.Layout(gtx,
									layout.Rigid(func(gtx C) D {
										return pg.theme.Body1((*pg.selectedAccount).TotalBalance).Layout(gtx)
									}),
									layout.Rigid(func(gtx C) D {

										inset := layout.Inset{
											Left: values.MarginPadding5,
											Top:  values.MarginPadding2,
										}
										return inset.Layout(gtx, func(gtx C) D {
											icon := common.icons.collapseIcon
											icon.Scale = 0.75
											return icon.Layout(gtx)
										})
									}),
								)
							})
						}),
					)
				})
			})
		})
	})
}

func (pg *sendPage) toSection(gtx layout.Context, common pageCommon) layout.Dimensions {
	return pg.pageSections(gtx, "To", func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Inset{
					Top:    values.MarginPadding10,
					Bottom: values.MarginPadding10,
				}.Layout(gtx, func(gtx C) D {
					return pg.destinationAddressEditor.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx C) D {
				if strings.Contains(pg.currencyValue, "USD") {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(0.45, func(gtx C) D {
							pg.leftAmountEditor.Hint = fmt.Sprintf("Amount (%s)", pg.leftExchangeValue)
							return pg.leftAmountEditor.Layout(gtx)
						}),
						layout.Flexed(0.1, func(gtx C) D {
							return layout.Inset{Top: values.MarginPadding10}.Layout(gtx, func(gtx C) D {
								return pg.currencySwap.Layout(gtx)
							})
						}),
						layout.Flexed(0.45, func(gtx C) D {
							pg.rightAmountEditor.Hint = fmt.Sprintf("Amount (%s)", pg.rightExchangeValue)
							return pg.rightAmountEditor.Layout(gtx)
						}),
					)
				}
				return pg.leftAmountEditor.Layout(gtx)
			}),
		)
	})
}

func (pg *sendPage) feeSection(gtx layout.Context) layout.Dimensions {
	collapsibleHeader := func(gtx C) D {
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				b := pg.theme.Body1(pg.leftTransactionFeeValue)
				b.Color = pg.theme.Color.Text
				return b.Layout(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				b := pg.theme.Body1(pg.rightTransactionFeeValue)
				b.Color = pg.theme.Color.Hint
				inset := layout.Inset{
					Left: values.MarginPadding5,
				}
				if strings.Contains(pg.currencyValue, "USD") {
					return inset.Layout(gtx, func(gtx C) D {
						return b.Layout(gtx)
					})
				}
				return layout.Dimensions{}
			}),
		)
	}

	collapsibleBody := func(gtx C) D {
		card := pg.theme.Card()
		card.Color = pg.theme.Color.Background
		inset := layout.Inset{
			Top: values.MarginPadding10,
		}
		return inset.Layout(gtx, func(gtx C) D {
			return card.Layout(gtx, func(gtx C) D {
				return layout.UniformInset(values.MarginPadding15).Layout(gtx, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return pg.contentRow(gtx, "Estimated time", "-", "")
						}),
						layout.Rigid(func(gtx C) D {
							inset := layout.Inset{
								Top:    values.MarginPadding5,
								Bottom: values.MarginPadding5,
							}
							return inset.Layout(gtx, func(gtx C) D {
								return pg.contentRow(gtx, "Estimated size", "-", "")
							})
						}),
						layout.Rigid(func(gtx C) D {
							return pg.contentRow(gtx, "Fee rate", "10 atoms/Byte", "")
						}),
					)
				})
			})
		})
	}
	return pg.pageSections(gtx, "Fee", func(gtx C) D {
		return pg.txFeeCollapsible.Layout(gtx, collapsibleHeader, collapsibleBody)
	})
}

func (pg *sendPage) balanceSection(gtx layout.Context, common pageCommon) layout.Dimensions {
	c := pg.theme.Card()
	c.Radius = decredmaterial.CornerRadius{NE: 0, NW: 0, SE: 0, SW: 0}
	return c.Layout(gtx, func(gtx C) D {
		return common.UniformPadding(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(0.6, func(gtx C) D {
					inset := layout.Inset{
						Right: values.MarginPadding15,
					}
					return inset.Layout(gtx, func(gtx C) D {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								inset := layout.Inset{
									Bottom: values.MarginPadding10,
								}
								return inset.Layout(gtx, func(gtx C) D {
									if strings.Contains(pg.currencyValue, "USD") {
										return pg.contentRow(gtx, "Total cost", pg.leftTotalCostValue+" "+pg.rightTotalCostValue, "")
									}
									return pg.contentRow(gtx, "Total cost", pg.leftTotalCostValue, "")
								})
							}),
							layout.Rigid(func(gtx C) D {
								return pg.contentRow(gtx, "Balance after send", pg.balanceAfterSendValue, "")
							}),
						)
					})
				}),
				layout.Flexed(0.3, func(gtx C) D {
					inset := layout.Inset{
						Top: values.MarginPadding2,
					}
					return inset.Layout(gtx, func(gtx C) D {
						return pg.nextButton.Layout(gtx)
					})
				}),
			)
		})
	})
}

func (pg *sendPage) walletAccountSection(gtx layout.Context, common pageCommon) layout.Dimensions {
	sections := func(gtx layout.Context, title string, body layout.Widget) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				txt := pg.theme.Body2(title)
				txt.Color = pg.theme.Color.Text
				inset := layout.Inset{
					Bottom: values.MarginPadding15,
				}
				return inset.Layout(gtx, func(gtx C) D {
					return txt.Layout(gtx)
				})
			}),
			layout.Rigid(body),
		)
	}

	w := []func(gtx C) D{
		func(gtx C) D {
			txt := pg.theme.H6("Sending account")
			txt.Color = pg.theme.Color.Text
			return txt.Layout(gtx)
		},
		func(gtx C) D {
			return pg.walletsList.Layout(gtx, len(common.info.Wallets), func(gtx C, i int) D {
				wn := common.info.Wallets[i].Name
				accounts := common.info.Wallets[i].Accounts
				wIndex := i

				pg.updateAcctDetailsButtons(&accounts)
				inset := layout.Inset{
					Bottom: values.MarginPadding10,
				}
				return inset.Layout(gtx, func(gtx C) D {
					return sections(gtx, wn, func(gtx C) D {
						return pg.accountsList.Layout(gtx, len(accounts), func(gtx C, x int) D {
							accountsName := accounts[x].Name
							totalBalance := accounts[x].TotalBalance
							spendable := dcrutil.Amount(accounts[x].SpendableBalance).String()
							aIndex := x

							click := pg.toAcctDetails[x]
							pointer.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Add(gtx.Ops)
							click.Add(gtx.Ops)
							pg.goToAcctDetails(gtx, common, &accounts[x], wIndex, aIndex, click)

							return pg.walletAccountsLayout(gtx, accountsName, totalBalance, spendable, common, wIndex, aIndex)
						})
					})
				})
			})
		},
	}

	return pg.walletAccountModal.Layout(gtx, w, 850)
}

func (pg *sendPage) walletAccountsLayout(gtx layout.Context, name, totalBal, spendableBal string, common pageCommon, wIndex, aIndex int) layout.Dimensions {
	accountIcon := common.icons.accountIcon
	accountIcon.Scale = 0.8

	inset := layout.Inset{
		Bottom: values.MarginPadding10,
	}
	return inset.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					layout.Flexed(0.1, func(gtx C) D {
						inset := layout.Inset{
							Right: values.MarginPadding10,
							Top:   values.MarginPadding15,
						}
						return inset.Layout(gtx, func(gtx C) D {
							return accountIcon.Layout(gtx)
						})
					}),
					layout.Flexed(0.8, func(gtx C) D {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								accountLabel := pg.theme.Body2(name)
								accountLabel.Color = pg.theme.Color.Text
								accountBalLabel := pg.theme.Body2(totalBal)
								accountBalLabel.Color = pg.theme.Color.Text
								return pg.accountTableLayout(gtx, accountLabel, accountBalLabel)
							}),
							layout.Rigid(func(gtx C) D {
								spendibleLabel := pg.theme.Body2("Spendable")
								spendibleLabel.Color = pg.theme.Color.Gray
								spendibleBalLabel := pg.theme.Body2(spendableBal)
								spendibleBalLabel.Color = pg.theme.Color.Gray
								return pg.accountTableLayout(gtx, spendibleLabel, spendibleBalLabel)
							}),
						)
					}),
					layout.Flexed(0.1, func(gtx C) D {
						inset := layout.Inset{
							Right: values.MarginPadding10,
							Top:   values.MarginPadding10,
						}

						// fmt.Println(*common.selectedWallet ,"==", wIndex)
						if *common.selectedWallet == wIndex {
							if *common.selectedAccount == aIndex {
								return layout.E.Layout(gtx, func(gtx C) D {
									return inset.Layout(gtx, func(gtx C) D {
										return common.icons.navigationCheck.Layout(gtx, values.MarginPadding20)
									})
								})
							}
						}
						return layout.Dimensions{}

					}),
				)
			}),
		)
	})
}

func (pg *sendPage) accountTableLayout(gtx layout.Context, leftLabel, rightLabel decredmaterial.Label) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			inset := layout.Inset{
				Top: values.MarginPadding2,
			}
			return inset.Layout(gtx, func(gtx C) D {
				return leftLabel.Layout(gtx)
			})
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.E.Layout(gtx, func(gtx C) D {
				return rightLabel.Layout(gtx)
			})
		}),
	)
}

func (pg *sendPage) updateAcctDetailsButtons(walAcct *[]wallet.Account) {
	if len(*walAcct) != len(pg.toAcctDetails) {
		for i := 0; i < len(*walAcct); i++ {
			pg.toAcctDetails = append(pg.toAcctDetails, &gesture.Click{})
		}
	}
}

func (pg *sendPage) goToAcctDetails(gtx layout.Context, common pageCommon, acct *wallet.Account, wIndex, aIndex int, click *gesture.Click) {
	for _, e := range click.Events(gtx) {
		if e.Type == gesture.TypeClick {
			fmt.Println(wIndex, "  ", aIndex)
			*pg.selectedAccount = acct
			// *common.selectedAccount = aIndex
			*common.selectedWallet = wIndex
			pg.selectedWallet = common.info.Wallets[wIndex]
			pg.isWalletAccountModalOpen = false
		}
	}
}

func (pg *sendPage) confirmationModal(gtx layout.Context, common pageCommon) layout.Dimensions {
	w := []func(gtx C) D{
		func(gtx C) D {
			return pg.theme.H6("Confim to send").Layout(gtx)
		},
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					icon := common.icons.sendIcon
					// icon.Scale = 0.8
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layout.Inset{Top: values.MarginPadding5, Right: values.MarginPadding10}.Layout(gtx, icon.Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return layoutBalance(gtx, "3.142320", common)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							icon := common.icons.navigationArrowForward
							icon.Color = pg.theme.Color.Gray
							return layout.Inset{Right: values.MarginPadding10}.Layout(gtx, func(gtx C) D {
								return icon.Layout(gtx, values.MarginPadding15)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return pg.theme.Body2("kjkjhbhcdxdfgcjhgkjnbv gfvj").Layout(gtx)
						}),
					)
				}),
			)
		},
		func(gtx C) D {
			pg.line.Width = gtx.Constraints.Max.X
			return pg.line.Layout(gtx)
		},
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return pg.contentRow(gtx, "Sending from", "Account name", "Selected wallet")
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Top: values.MarginPadding10, Bottom: values.MarginPadding10}.Layout(gtx, func(gtx C) D {
						return pg.contentRow(gtx, "Fee", "345678", "")
					})
				}),
				layout.Rigid(func(gtx C) D {
					return pg.contentRow(gtx, "Total cost", "345678", "")
				}),
			)
		},
		func(gtx C) D {
			return pg.passwordEditor.Layout(gtx)
		},
		func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					icon := common.icons.actionInfo
					icon.Color = pg.theme.Color.Gray
					return layout.Inset{Right: values.MarginPadding10}.Layout(gtx, func(gtx C) D {
						return icon.Layout(gtx, values.MarginPadding20)
					})
				}),
				layout.Rigid(func(gtx C) D {
					return pg.theme.Body2("Your DCR will be sent after this step.").Layout(gtx)
				}),
			)
		},
		func(gtx C) D {
			return layout.E.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						inset := layout.Inset{
							Left: values.MarginPadding5,
						}
						return inset.Layout(gtx, func(gtx C) D {
							return pg.closeConfirmationModalButton.Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx C) D {
						pg.confirmButton.Text = "4.5456566 DCR"
						return pg.confirmButton.Layout(gtx)
					}),
				)
			})
		},
	}

	return pg.confirmModal.Layout(gtx, w, 1000)
}

func (pg *sendPage) pageSections(gtx layout.Context, title string, body layout.Widget) layout.Dimensions {
	return pg.theme.Card().Layout(gtx, func(gtx C) D {
		return layout.UniformInset(values.MarginPadding15).Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							txt := pg.theme.Body1(title)
							txt.Color = pg.theme.Color.Text
							inset := layout.Inset{
								Bottom: values.MarginPadding10,
							}
							return inset.Layout(gtx, func(gtx C) D {
								return txt.Layout(gtx)
							})
						}),
						layout.Flexed(1, func(gtx C) D {
							if title == "To" {
								return layout.E.Layout(gtx, func(gtx C) D {
									return pg.accountSwitch.Layout(gtx)
								})
							}
							return layout.Dimensions{}
						}),
					)
				}),
				layout.Rigid(body),
			)
		})
	})
}

func (pg *sendPage) contentRow(gtx layout.Context, leftValue, rightValue, walletName string) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			txt := pg.theme.Body2(leftValue)
			txt.Color = pg.theme.Color.Gray
			return txt.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.E.Layout(gtx, func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						txt := pg.theme.Body2(rightValue)
						txt.Color = pg.theme.Color.Text
						return txt.Layout(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						if walletName != "" {
							card := pg.theme.Card()
							card.Radius = decredmaterial.CornerRadius{
								NE: 0,
								NW: 0,
								SE: 0,
								SW: 0,
							}
							card.Color = pg.theme.Color.Background
							inset := layout.Inset{
								Left: values.MarginPadding5,
							}
							return inset.Layout(gtx, func(gtx C) D {
								return card.Layout(gtx, func(gtx C) D {
									return layout.UniformInset(values.MarginPadding2).Layout(gtx, func(gtx C) D {
										txt := pg.theme.Caption(walletName)
										txt.Color = pg.theme.Color.Gray
										return txt.Layout(gtx)
									})
								})
							})
						}
						return layout.Dimensions{}
					}),
				)
			})
		}),
	)
}

func (pg *sendPage) validate() bool {
	isAmountValid := pg.validateLeftAmount()
	if pg.rightAmountEditor.Editor.Focused() {
		isAmountValid = pg.validateRightAmount()
	}
	if !pg.validateDestinationAddress() || !isAmountValid || pg.calculateErrorText != "" {
		pg.nextButton.Background = pg.theme.Color.Hint
		return false
	}

	pg.nextButton.Background = pg.theme.Color.Primary
	return true
}

func (pg *sendPage) validateDestinationAddress() bool {
	if pg.inputsNotEmpty(pg.destinationAddressEditor.Editor) {
		isValid, _ := pg.wallet.IsAddressValid(pg.destinationAddressEditor.Editor.Text())
		if !isValid {
			pg.destinationAddressEditor.SetError("invalid address")
			return false
		}

		pg.destinationAddressEditor.SetError("")
		return true
	}
	pg.destinationAddressEditor.SetError("")
	return false
}

func (pg *sendPage) validateLeftAmount() bool {
	if pg.inputsNotEmpty(pg.leftAmountEditor.Editor) {
		_, err := strconv.ParseFloat(pg.leftAmountEditor.Editor.Text(), 64)
		if err != nil {
			return false
		}

		return true
	}

	return false
}

func (pg *sendPage) validateRightAmount() bool {
	if pg.inputsNotEmpty(pg.rightAmountEditor.Editor) {
		_, err := strconv.ParseFloat(pg.rightAmountEditor.Editor.Text(), 64)
		if err != nil {
			return false
		}

		return true
	}

	return false
}

func (pg *sendPage) inputsNotEmpty(editors ...*widget.Editor) bool {
	for _, e := range editors {
		if e.Text() == "" {
			pg.nextButton.Background, pg.confirmButton.Background = pg.theme.Color.Hint, pg.theme.Color.Hint
			return false
		}
	}
	pg.nextButton.Background, pg.confirmButton.Background = pg.theme.Color.Primary, pg.theme.Color.Primary
	return true
}

func (pg *sendPage) calculateValues() {
	defaultLeftValues := fmt.Sprintf("- %s", "DCR")
	defaultRightValues := "($ -)"

	pg.leftTransactionFeeValue = defaultLeftValues
	pg.rightTransactionFeeValue = defaultRightValues

	pg.leftTotalCostValue = defaultLeftValues
	pg.rightTotalCostValue = defaultRightValues
	pg.calculateErrorText = ""

	if reflect.DeepEqual(pg.txAuthor, &dcrlibwallet.TxAuthor{}) || !pg.validate() {
		return
	}

	pg.inputAmount, _ = strconv.ParseFloat(pg.leftAmountEditor.Editor.Text(), 64)
	if strings.Contains(pg.currencyValue, "USD") && pg.rightAmountEditor.Editor.Focused() {
		pg.inputAmount, _ = strconv.ParseFloat(pg.rightAmountEditor.Editor.Text(), 64)
	}

	if strings.Contains(pg.currencyValue, "USD") && pg.LastTradeRate != "" {
		pg.usdExchangeRate, _ = strconv.ParseFloat(pg.LastTradeRate, 64)
		pg.amountUSDtoDCR = pg.inputAmount / pg.usdExchangeRate
		pg.amountDCRtoUSD = pg.inputAmount * pg.usdExchangeRate
	}

	switch {
	case pg.leftExchangeValue == "USD" && pg.LastTradeRate != "" && pg.leftAmountEditor.Editor.Focused():
		pg.rightAmountEditor.Editor.SetText(fmt.Sprintf("%f", pg.amountUSDtoDCR))
		pg.setDestinationAddr(pg.amountUSDtoDCR)
	case pg.leftExchangeValue == "USD" && pg.LastTradeRate != "" && pg.rightAmountEditor.Editor.Focused():
		pg.leftAmountEditor.Editor.SetText(fmt.Sprintf("%f", pg.amountDCRtoUSD))
		pg.setDestinationAddr(pg.inputAmount)
	case pg.leftExchangeValue == "DCR" && pg.LastTradeRate != "" && pg.rightAmountEditor.Editor.Focused():
		pg.leftAmountEditor.Editor.SetText(fmt.Sprintf("%f", pg.amountUSDtoDCR))
		pg.setDestinationAddr(pg.amountUSDtoDCR)
	case pg.leftExchangeValue == "DCR" && pg.LastTradeRate != "" && pg.leftAmountEditor.Editor.Focused():
		pg.rightAmountEditor.Editor.SetText(fmt.Sprintf("%f", pg.amountDCRtoUSD))
		pg.setDestinationAddr(pg.inputAmount)
	default:
		pg.rightAmountEditor.Editor.SetText(pg.leftAmountEditor.Editor.Text())
		if pg.rightAmountEditor.Editor.Focused() {
			pg.leftAmountEditor.Editor.SetText(pg.rightAmountEditor.Editor.Text())
		}
		pg.setDestinationAddr(pg.inputAmount)
	}

	if pg.amountAtoms == 0 {
		return
	}

	pg.txFee = pg.getTxFee()
	if pg.txFee == 0 {
		return
	}

	pg.totalCostDCR = pg.txFee + pg.amountAtoms

	pg.updateDefaultValues()
	// pg.balanceAfterSend(false)
}

func (pg *sendPage) setDestinationAddr(sendAmount float64) {
	pg.amountErrorText = ""
	amount, err := dcrutil.NewAmount(sendAmount)
	if err != nil {
		pg.feeEstimationError(err.Error(), "amount")
		return
	}

	pg.amountAtoms = int64(amount)
	pg.txAuthor.RemoveSendDestination(0)
	pg.txAuthor.AddSendDestination(pg.destinationAddressEditor.Editor.Text(), pg.amountAtoms, false)
}

func (pg *sendPage) amountValues() amountValue {
	txFeeValueUSD := dcrutil.Amount(pg.txFee).ToCoin() * pg.usdExchangeRate
	switch {
	case pg.leftExchangeValue == "USD" && pg.LastTradeRate != "":
		return amountValue{
			inactiveInputAmount:      dcrutil.Amount(pg.amountAtoms).String(),
			leftTransactionFeeValue:  fmt.Sprintf("%f USD", txFeeValueUSD),
			rightTransactionFeeValue: fmt.Sprintf("(%s)", dcrutil.Amount(pg.txFee).String()),
			leftTotalCostValue:       fmt.Sprintf("%s USD", strconv.FormatFloat(pg.inputAmount+txFeeValueUSD, 'f', 7, 64)),
			rightTotalCostValue:      fmt.Sprintf("(%s )", dcrutil.Amount(pg.totalCostDCR).String()),
		}
	case pg.leftExchangeValue == "DCR" && pg.LastTradeRate != "":
		return amountValue{
			inactiveInputAmount:      fmt.Sprintf("%s USD", strconv.FormatFloat(pg.amountDCRtoUSD, 'f', 2, 64)),
			leftTransactionFeeValue:  dcrutil.Amount(pg.txFee).String(),
			rightTransactionFeeValue: fmt.Sprintf("(%f USD)", txFeeValueUSD),
			leftTotalCostValue:       dcrutil.Amount(pg.totalCostDCR).String(),
			rightTotalCostValue:      fmt.Sprintf("(%s USD)", strconv.FormatFloat(pg.amountDCRtoUSD+txFeeValueUSD, 'f', 7, 64)),
		}
	default:
		return amountValue{
			inactiveInputAmount:     "Exchange rate not fetched",
			leftTransactionFeeValue: dcrutil.Amount(pg.txFee).String(),
			leftTotalCostValue:      dcrutil.Amount(pg.totalCostDCR).String(),
		}
	}
}

func (pg *sendPage) getTxFee() int64 {
	// calculate transaction fee
	feeAndSize, err := pg.txAuthor.EstimateFeeAndSize()
	if err != nil {
		pg.feeEstimationError(err.Error(), "fee")
		return 0
	}

	return feeAndSize.Fee.AtomValue
}

func (pg *sendPage) balanceAfterSend(isInputAmountEmpty bool) {
	pg.remainingBalance = 0
	if isInputAmountEmpty {
		pg.remainingBalance = (*pg.selectedAccount).SpendableBalance
	} else {
		pg.remainingBalance = (*pg.selectedAccount).SpendableBalance - pg.totalCostDCR
	}
	pg.balanceAfterSendValue = dcrutil.Amount(pg.remainingBalance).String()
}

func (pg *sendPage) feeEstimationError(err, errorPath string) {
	if err == "insufficient_balance" {
		pg.amountErrorText = "Not enough funds"
	}
	if strings.Contains(err, "invalid amount") {
		pg.amountErrorText = "Invalid amount"
	}
	pg.calculateErrorText = fmt.Sprintf("error estimating transaction %s: %s", errorPath, err)
}

func (pg *sendPage) watchForBroadcastResult(c pageCommon) {
	if pg.broadcastResult == nil {
		return
	}

	if pg.broadcastResult.TxHash != "" {
		if pg.remainingBalance != -1 {
			pg.spendableBalance = pg.remainingBalance
		}
		pg.remainingBalance = -1
		c.Notify("Transaction Sent", true)

		pg.destinationAddressEditor.Editor.SetText("")
		pg.sendAmountEditor.Editor.SetText("")
		pg.isConfirmationModalOpen = false
		pg.isBroadcastingTransaction = false
		pg.broadcastResult.TxHash = ""
		pg.calculateValues()
		(*pg.unspentOutputsSelected)[pg.selectedWallet.ID][(*pg.selectedAccount).Number] = make(map[string]*wallet.UnspentOutput)
	}
}

// handleEditorChange handles changes on the editor and adjust its width of the send amount input field
// it also updates the DCR - USD exchange rate value
func (pg *sendPage) handleEditorChange(evt widget.EditorEvent) {
	switch evt.(type) {
	case widget.ChangeEvent:
		go pg.wallet.GetUSDExchangeValues(&pg, pg.errChan)
		pg.calculateValues()
	}
}

func (pg *sendPage) updateDefaultValues() {
	// if strings.Contains(pg.currencyValue, "USD") {
	// 	if pg.rightAmountEditor.Editor.Focused() {
	// 		// pg.rightAmountEditor.Editor.Move(pg.rightAmountEditor.Editor.NumLines())
	// 		pg.leftAmountEditor.Editor.SetText(pg.rightAmountEditor.Editor.Text())
	// 		return
	// 	}
	// 	if pg.leftAmountEditor.Editor.Focused() {
	// 		// pg.leftAmountEditor.Editor.Move(pg.leftAmountEditor.Editor.NumLines())
	// 		pg.rightAmountEditor.Editor.SetText(pg.leftAmountEditor.Editor.Text())
	// 		return
	// 	}
	// }

	v := pg.amountValues()
	pg.activeTotalAmount = pg.leftExchangeValue
	pg.leftTransactionFeeValue = v.leftTransactionFeeValue
	pg.rightTransactionFeeValue = v.rightTransactionFeeValue
	pg.leftTotalCostValue = v.leftTotalCostValue
	pg.rightTotalCostValue = v.rightTotalCostValue
}

func (pg *sendPage) Handle(c pageCommon) {
	if len(c.info.Wallets) == 0 {
		return
	}

	if pg.LastTradeRate == "" && pg.count == 0 {
		pg.count = 1
		pg.calculateValues()
	}

	if (pg.LastTradeRate != "" && pg.count == 0) || (pg.LastTradeRate != "" && pg.count == 1) {
		pg.count = 2
		pg.calculateValues()
	}

	if pg.exchangeErr != "" {
		c.Notify(pg.exchangeErr, false)
	}

	if c.subPageBackButton.Button.Clicked() {
		*c.page = PageOverview
	}

	if c.subPageInfoButton.Button.Clicked() {
		go func() {
			c.modalReceiver <- &modalLoad{
				template:   SendInfoTemplate,
				title:      "Send DCR",
				cancel:     c.closeModal,
				cancelText: "Got it",
			}
		}()
	}

	for pg.moreOption.Button.Clicked() {
		pg.isMoreOption = !pg.isMoreOption
	}

	if pg.fromAccountBtn.Clicked() {
		pg.isWalletAccountModalOpen = true
	}

	pg.currencyValue = pg.wallet.ReadStringConfigValueForKey(dcrlibwallet.CurrencyConversionConfigKey)
	if pg.currencyValue == "" {
		pg.currencyValue = "None"
	}

	for range pg.destinationAddressEditor.Editor.Events() {
		pg.calculateValues()
	}

	for pg.currencySwap.Button.Clicked() {
		if pg.leftExchangeValue == "DCR" {
			pg.leftExchangeValue = "USD"
			pg.rightExchangeValue = "DCR"
		} else {
			pg.leftExchangeValue = "DCR"
			pg.rightExchangeValue = "USD"
		}

		pg.calculateValues()
	}

	for _, evt := range pg.leftAmountEditor.Editor.Events() {
		if pg.leftAmountEditor.Editor.Focused() {
			pg.handleEditorChange(evt)
		}
	}

	for _, evt := range pg.rightAmountEditor.Editor.Events() {
		if pg.rightAmountEditor.Editor.Focused() {
			pg.handleEditorChange(evt)
		}
	}

	if pg.calculateErrorText != "" {
		pg.leftAmountEditor.LineColor, pg.leftAmountEditor.TitleLabelColor = pg.theme.Color.Danger, pg.theme.Color.Danger
		pg.rightAmountEditor.LineColor, pg.rightAmountEditor.TitleLabelColor = pg.theme.Color.Danger, pg.theme.Color.Danger
		c.Notify(pg.calculateErrorText, false)
	} else {
		pg.leftAmountEditor.LineColor, pg.leftAmountEditor.TitleLabelColor = pg.theme.Color.Hint, pg.theme.Color.Gray
		pg.rightAmountEditor.LineColor, pg.rightAmountEditor.TitleLabelColor = pg.theme.Color.Hint, pg.theme.Color.Gray
	}

	if pg.shouldInitializeTxAuthor {
		pg.shouldInitializeTxAuthor = false
		pg.leftAmountEditor.Editor.SetText("")
		pg.rightAmountEditor.Editor.SetText("")
		pg.calculateErrorText = ""
		c.wallet.CreateTransaction(pg.selectedWallet.ID, (*pg.selectedAccount).Number, pg.txAuthorErrChan)
	}

	// // pg.validate(true)
	// pg.watchForBroadcastResult(c)

	// if pg.isBroadcastingTransaction {
	// 	col := pg.theme.Color.Gray
	// 	col.A = 150

	// 	pg.nextButton.Text = "Sending..."
	// 	pg.nextButton.Background = col
	// } else {
	// 	pg.nextButton.Text = "Next"
	// 	pg.nextButton.Background = pg.theme.Color.Primary
	// }

	// for pg.nextButton.Button.Clicked() {
	// 	// if pg.validate(false) && pg.calculateErrorText == "" {
	// 		pg.isConfirmationModalOpen = true
	// 	// }
	// }

	// for pg.confirmButton.Button.Clicked() {
	// 	pg.isConfirmationModalOpen = false
	// 	pg.isPasswordModalOpen = true
	// }

	// for pg.closeConfirmationModalButton.Button.Clicked() {
	// 	pg.isConfirmationModalOpen = false
	// }

	// for pg.maxButton.Button.Clicked() {
	// 	pg.activeExchange = "DCR"
	// 	amountMax, err := pg.txAuthor.EstimateMaxSendAmount()
	// 	if err != nil {
	// 		return
	// 	}
	// 	pg.sendAmountEditor.Editor.SetText(fmt.Sprintf("%.10f", amountMax.DcrValue))
	// 	pg.calculateValues()
	// }

	// if pg.destinationAddressEditor.Editor.Len() == 0 || pg.leftAmountEditor.Editor.Len() == 0 || pg.leftAmountEditor.Editor.Len() == 0 {
	// 	pg.balanceAfterSend(true)
	// }

	for pg.nextButton.Button.Clicked() {
		pg.isConfirmationModalOpen = true
	}

	for pg.closeConfirmationModalButton.Button.Clicked() {
		pg.isConfirmationModalOpen = false
	}

	select {
	case err := <-pg.txAuthorErrChan:
		pg.calculateErrorText = err.Error()
		c.Notify(pg.calculateErrorText, false)
	case err := <-pg.broadcastErrChan:
		c.Notify(err.Error(), false)

		if err.Error() == invalidPassphraseError {
			time.AfterFunc(time.Second*3, func() {
				pg.isConfirmationModalOpen = true
			})
		}
		pg.isBroadcastingTransaction = false
	case err := <-pg.errChan:
		errMsg := err.Error()
		if strings.Contains(err.Error(), "host") {
			errMsg = "Could fetch exchange: no such host"
		}
		c.Notify(errMsg, false)
	default:
	}
}
