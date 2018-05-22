package client

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"text/tabwriter"

	"github.com/bgentry/speakeasy"
	"github.com/spf13/cobra"

	"github.com/rivine/rivine/api"
	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/types"
)

// have to be called prior to being able to use wallet cmds
func createWalletCommands() {
	walletCmd = &cobra.Command{
		Use:   "wallet",
		Short: "Perform wallet actions",
		Long: `Generate a new address, send coins to another wallet, or view info about the wallet.

` + _CurrencyConvertor.CoinHelp(),
		Run: Wrap(walletbalancecmd),
	}

	walletBlockStakeStatCmd = &cobra.Command{
		Use:   "blockstakestat",
		Short: "Get the stats of the blockstake",
		Long:  "Gives all the statistical info of the blockstake.",
		Run:   Wrap(walletblockstakestatcmd),
	}

	walletAddressCmd = &cobra.Command{
		Use:   "address",
		Short: "Get a new wallet address",
		Long:  "Generate a new wallet address from the wallet's primary seed.",
		Run:   Wrap(walletaddresscmd),
	}

	walletAddressesCmd = &cobra.Command{
		Use:   "addresses",
		Short: "List all addresses",
		Long:  "List all addresses that have been generated by the wallet",
		Run:   Wrap(walletaddressescmd),
	}

	walletInitCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize and encrypt a new wallet",
		Long:  `Generate a new wallet from a randomly generated seed, and encrypt it.`,
		Run:   Wrap(walletinitcmd),
	}

	walletRecoverCmd = &cobra.Command{
		Use:   "recover",
		Short: "Recover and encrypt a new wallet",
		Long:  `Recover a wallet from the given mnemonic, to be used as primary seed, and encrypt it.`,
		Run:   Wrap(walletrecovercmd),
	}

	walletLoadCmd = &cobra.Command{
		Use:   "load",
		Short: "Load a wallet seed",
		// Run field is not set, as the load command itself is not a valid command.
		// A subcommand must be provided.
	}

	walletLoadSeedCmd = &cobra.Command{
		Use:   `seed`,
		Short: "Add a seed to the wallet",
		Long:  "Uses the given password to create a new wallet with that as the primary seed",
		Run:   Wrap(walletloadseedcmd),
	}

	walletLockCmd = &cobra.Command{
		Use:   "lock",
		Short: "Lock the wallet",
		Long:  "Lock the wallet, preventing further use",
		Run:   Wrap(walletlockcmd),
	}

	walletSeedsCmd = &cobra.Command{
		Use:   "seeds",
		Short: "Retrieve information about your seeds",
		Long:  "Retrieves the current seed, how many addresses are remaining, and the rest of your seeds from the wallet",
		Run:   Wrap(walletseedscmd),
	}

	walletSendCmd = &cobra.Command{
		Use:   "send",
		Short: "Send either coins or blockstakes",
		Long:  "Send either coins or blockstakes",
		// Run field is not set, as the load command itself is not a valid command.
		// A subcommand must be provided.
	}

	walletSendCoinsCmd = &cobra.Command{
		Use:   "coins <dest>|<rawCondition> <amount> [<dest>|<rawCondition> <amount>]...",
		Short: "Send coins one or multiple addresses.",
		Long: `Send coins to one or multiple addresses.
Each 'dest' must be a 78-byte hexadecimal address (Unlock Hash),
instead of an unlockHash, you can also give a JSON-encoded UnlockCondition directly,
giving you more control and options over how exactly the block stake is to be unlocked.

` + _CurrencyConvertor.CoinArgDescription("amount") + `

Miner fees will be added on top of the given amount automatically.

`,
		Run: walletsendcoinscmd,
	}

	walletSendBlockStakesCmd = &cobra.Command{
		Use:   "blockstakes <dest>|<rawCondition> <amount> [<dest>|<rawCondition> <amount>]..",
		Short: "Send blockstakes to one or multiple addresses",
		Long: `Send blockstakes to one or multiple addresses.
Each 'dest' must be a 78-byte hexadecimal address (Unlock Hash),
instead of an unlockHash, you can also give a JSON-encoded UnlockCondition directly,
giving you more control and options over how exactly the block stake is to be unlocked.

Miner fees (expressed in ` + _CurrencyCoinUnit + `) will be added on top automatically.

`,
		Run: walletsendblockstakescmd,
	}

	walletRegisterDataCmd = &cobra.Command{
		Use:   "registerdata <namespace> <data> <dest>",
		Short: "Register data on the blockchain",
		Long:  "Register data on the blockcahin by sending a minimal transaction to the destination address, and including the data in the transaction",
		Run:   Wrap(walletregisterdatacmd),
	}

	walletBalanceCmd = &cobra.Command{
		Use:   "balance",
		Short: "View wallet balance",
		Long:  "View wallet balance, including confirmed and unconfirmed coins and blockstakes.",
		Run:   Wrap(walletbalancecmd),
	}

	walletTransactionsCmd = &cobra.Command{
		Use:   "transactions",
		Short: "View transactions",
		Long:  "View transactions related to addresses spendable by the wallet, providing a net flow of coins and blockstakes for each transaction",
		Run:   Wrap(wallettransactionscmd),
	}

	walletUnlockCmd = &cobra.Command{
		Use:   `unlock`,
		Short: "Unlock the wallet",
		Long:  "Decrypt and load the wallet into memory",
		Run:   Wrap(walletunlockcmd),
	}

	walletSendTxnCmd = &cobra.Command{
		Use:   "transaction <txnjson>",
		Short: "Publish a raw transaction",
		Long:  "Publish a raw transasction. The transaction must be given in json format. The inputs don't need to be related to the current wallet",
		Run:   Wrap(walletsendtxncmd),
	}
}

// still need to be initialized using createWalletCommands
var (
	walletCmd                *cobra.Command
	walletBlockStakeStatCmd  *cobra.Command
	walletAddressCmd         *cobra.Command
	walletAddressesCmd       *cobra.Command
	walletInitCmd            *cobra.Command
	walletRecoverCmd         *cobra.Command
	walletLoadCmd            *cobra.Command
	walletLoadSeedCmd        *cobra.Command
	walletLockCmd            *cobra.Command
	walletSendCmd            *cobra.Command
	walletSeedsCmd           *cobra.Command
	walletSendCoinsCmd       *cobra.Command
	walletSendBlockStakesCmd *cobra.Command
	walletRegisterDataCmd    *cobra.Command
	walletBalanceCmd         *cobra.Command
	walletTransactionsCmd    *cobra.Command
	walletUnlockCmd          *cobra.Command
	walletSendTxnCmd         *cobra.Command
)

// walletaddresscmd fetches a new address from the wallet that will be able to
// receive coins.
func walletaddresscmd() {
	addr := new(api.WalletAddressGET)
	err := _DefaultClient.httpClient.GetAPI("/wallet/address", addr)
	if err != nil {
		Die("Could not generate new address:", err)
	}
	fmt.Printf("Created new address: %s\n", addr.Address)
}

// walletaddressescmd fetches the list of addresses that the wallet knows.
func walletaddressescmd() {
	addrs := new(api.WalletAddressesGET)
	err := _DefaultClient.httpClient.GetAPI("/wallet/addresses", addrs)
	if err != nil {
		Die("Failed to fetch addresses:", err)
	}
	for _, addr := range addrs.Addresses {
		fmt.Println(addr)
	}
}

// walletinitcmd encrypts the wallet with the given password
func walletinitcmd() {
	var er api.WalletInitPOST

	fmt.Println("You have to provide a passphrase!")
	fmt.Println("If you have an existing mnemonic you can use the recover wallet command instead.")

	passphrase, err := speakeasy.Ask("Wallet passphrase: ")
	if err != nil {
		Die("Reading passphrase failed:", err)
	}
	if passphrase == "" {
		Die("passphrase is required and cannot be empty")
	}

	repassphrase, err := speakeasy.Ask("Reenter passphrase: ")
	if err != nil {
		Die("Reading passphrase failed:", err)
	}

	if repassphrase != passphrase {
		Die("Given passphrases do not match !!")
	}

	qs := fmt.Sprintf("passphrase=%s", passphrase)

	err = _DefaultClient.httpClient.PostResp("/wallet/init", qs, &er)
	if err != nil {
		Die("Error when encrypting wallet:", err)
	}

	fmt.Printf("Mnemonic of primary seed:\n%s\n\n", er.PrimarySeed)
	fmt.Printf("Wallet encrypted with given passphrase\n")
}

// walletrecovercmd encrypts the wallet with the given password,
// recovering a wallet for the given menmeonic to be used as primary seed.
func walletrecovercmd() {
	var er api.WalletInitPOST

	fmt.Println("You have to provide a passphrase and existing mnemonic!")
	fmt.Println("If you have no existing mnemonic use the init wallet command instead!")

	passphrase, err := speakeasy.Ask("Wallet passphrase: ")
	if err != nil {
		Die("Reading passphrase failed:", err)
	}
	if passphrase == "" {
		Die("passphrase is required and cannot be empty")
	}

	repassphrase, err := speakeasy.Ask("Reenter passphrase: ")
	if err != nil {
		Die("Reading passphrase failed:", err)
	}

	if repassphrase != passphrase {
		Die("Given passphrases do not match !!")
	}

	mnemonic, err := speakeasy.Ask("Enter existing mnemonic to be used as primary seed: ")
	if err != nil {
		Die("Reading mnemonic failed:", err)
	}

	seed, err := modules.InitialSeedFromMnemonic(mnemonic)
	if err != nil {
		Die("Invalid mnemonic given:", err)
	}

	qs := fmt.Sprintf("passphrase=%s&seed=%s", passphrase, seed)

	err = _DefaultClient.httpClient.PostResp("/wallet/init", qs, &er)
	if err != nil {
		Die("Error when encrypting wallet:", err)
	}

	if er.PrimarySeed != mnemonic {
		Die("Wallet was created, but returned primary seed mnemonic was unexpected:\n\n" + er.PrimarySeed)
	}

	fmt.Printf("Mnemonic of primary seed:\n%s\n\n", er.PrimarySeed)
	fmt.Printf("Wallet encrypted with given passphrase\n")
}

// Wwlletloadseedcmd adds a seed to the wallet's list of seeds
func walletloadseedcmd() {
	passphrase, err := speakeasy.Ask("Wallet passphrase: ")
	if err != nil {
		Die("Reading passphrase failed:", err)
	}
	mnemonic, err := speakeasy.Ask("New Mnemonic: ")
	if err != nil {
		Die("Reading seed failed:", err)
	}
	qs := fmt.Sprintf("passphrase=%s&mnemonic=%s", passphrase, mnemonic)
	err = _DefaultClient.httpClient.Post("/wallet/seed", qs)
	if err != nil {
		Die("Could not add seed:", err)
	}
	fmt.Println("Added Key")
}

// walletlockcmd locks the wallet
func walletlockcmd() {
	err := _DefaultClient.httpClient.Post("/wallet/lock", "")
	if err != nil {
		Die("Could not lock wallet:", err)
	}
}

// walletseedscmd returns the current seed {
func walletseedscmd() {
	var seedInfo api.WalletSeedsGET
	err := _DefaultClient.httpClient.GetAPI("/wallet/seeds", &seedInfo)
	if err != nil {
		Die("Error retrieving the current seed:", err)
	}
	fmt.Printf("Primary Seed: %s\n"+
		"Addresses Remaining %d\n"+
		"All Seeds:\n", seedInfo.PrimarySeed, seedInfo.AddressesRemaining)
	for _, seed := range seedInfo.AllSeeds {
		fmt.Println(seed)
	}
}

// walletsendcoinscmd sends siacoins to one or multiple destination addresses.
func walletsendcoinscmd(cmd *cobra.Command, args []string) {
	pairs, err := parsePairedOutputs(args)
	if err != nil {
		cmd.UsageFunc()(cmd)
		Die(err)
	}

	body := api.WalletCoinsPOST{
		CoinOutputs: make([]types.CoinOutput, len(pairs)),
	}
	for i, pair := range pairs {
		body.CoinOutputs[i] = types.CoinOutput{
			Value:     pair.Value,
			Condition: pair.Condition,
		}
	}

	bytes, err := json.Marshal(&body)
	if err != nil {
		Die("Failed to JSON Marshal the input body:", err)
	}
	err = _DefaultClient.httpClient.Post("/wallet/coins", string(bytes))
	if err != nil {
		Die("Could not send coins:", err)
	}
	for _, co := range body.CoinOutputs {
		fmt.Printf("Sent %s to %s\n", _CurrencyConvertor.ToCoinStringWithUnit(co.Value), co.Condition.UnlockHash())
	}
}

// walletsendblockstakescmd sends block stakes to one or multiple destination addresses.
func walletsendblockstakescmd(cmd *cobra.Command, args []string) {
	pairs, err := parsePairedOutputs(args)
	if err != nil {
		cmd.UsageFunc()(cmd)
		Die(err)
	}

	body := api.WalletBlockStakesPOST{
		BlockStakeOutputs: make([]types.BlockStakeOutput, len(pairs)),
	}
	for i, pair := range pairs {
		body.BlockStakeOutputs[i] = types.BlockStakeOutput{
			Value:     pair.Value,
			Condition: pair.Condition,
		}
	}

	bytes, err := json.Marshal(&body)
	if err != nil {
		Die("Failed to JSON Marshal the input body:", err)
	}
	err = _DefaultClient.httpClient.Post("/wallet/blockstakes", string(bytes))
	if err != nil {
		Die("Could not send block stakes:", err)
	}
	for _, bo := range body.BlockStakeOutputs {
		fmt.Printf("Sent %s BS to %s\n", bo.Value, bo.Condition.UnlockHash())
	}
}

type outputPair struct {
	Condition types.UnlockConditionProxy
	Value     types.Currency
}

func parsePairedOutputs(args []string) (pairs []outputPair, err error) {
	argn := len(args)
	if argn < 2 {
		err = errors.New("not enough arguments, at least 2 required")
		return
	}
	if argn%2 != 0 {
		err = errors.New("arguments have to be given in pairs of '<dest>|<rawCondition>'+'<value>'")
		return
	}

	for i := 0; i < argn; i += 2 {
		// parse value first, as it's the one without any possibility of ambiguity
		var pair outputPair
		pair.Value, err = _CurrencyConvertor.ParseCoinString(args[i+1])
		if err != nil {
			err = fmt.Errorf("failed to parse amount/value for output #%d: %v", i/2, err)
			return
		}

		// try to parse it as an unlock hash
		var uh types.UnlockHash
		err = uh.LoadString(args[i])
		if err == nil {
			// parsing as an unlock hash was succesfull, store the pair and continue to the next pair
			pair.Condition = types.NewCondition(types.NewUnlockHashCondition(uh))
			pairs = append(pairs, pair)
			continue
		}

		// try to parse it as a JSON-encoded unlock condition
		err = pair.Condition.UnmarshalJSON([]byte(args[i]))
		if err != nil {
			err = fmt.Errorf("condition has to be UnlockHash or JSON-encoded UnlockCondition, output #%d's was neither", i/2)
			return
		}
		pairs = append(pairs, pair)
	}
	return
}

// walletregisterdatacmd registers data on the blockchain by making a minimal transaction to the designated address
// and includes the data in the transaction
func walletregisterdatacmd(namespace, dest, data string) {
	encodedData := base64.StdEncoding.EncodeToString([]byte(namespace + data))
	err := _DefaultClient.httpClient.Post("/wallet/data",
		fmt.Sprintf("destination=%s&data=%s", dest, encodedData))
	if err != nil {
		Die("Could not register data:", err)
	}
	fmt.Printf("Registered data to %s\n", dest)
}

// walletblockstakestatcmd gives all statistical info of blockstake
func walletblockstakestatcmd() {
	bsstat := new(api.WalletBlockStakeStatsGET)
	err := _DefaultClient.httpClient.GetAPI("/wallet/blockstakestats", bsstat)
	if err != nil {
		Die("Could not gen blockstake info:", err)
	}
	fmt.Printf("BlockStake stats:\n")
	fmt.Printf("Total active Blockstake is %v\n", bsstat.TotalActiveBlockStake)
	fmt.Printf("This account has %v Blockstake\n", bsstat.TotalBlockStake)
	fmt.Printf("%v of last %v Blocks created (theoretically %v)\n", bsstat.TotalBCLast1000, bsstat.BlockCount, bsstat.TotalBCLast1000t)

	fmt.Printf("containing %v fee \n",
		_CurrencyConvertor.ToCoinStringWithUnit(bsstat.TotalFeeLast1000))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintln(w, "state\t#BlockStake\tUTXO hash\t")

	for i, BSstate := range bsstat.BlockStakeState {
		state := "active"
		if BSstate == 0 {
			state = "not active"
		}
		fmt.Fprintf(w, "%v\t%v\t%v\t\n", state, bsstat.BlockStakeNumOf[i], bsstat.BlockStakeUTXOAddress[i])
	}
	w.Flush()
}

// walletbalancecmd retrieves and displays information about the wallet.
func walletbalancecmd() {
	status := new(api.WalletGET)
	err := _DefaultClient.httpClient.GetAPI("/wallet", status)
	if err != nil {
		Die("Could not get wallet status:", err)
	}
	encStatus := "Unencrypted"
	if status.Encrypted {
		encStatus = "Encrypted"
	}
	if !status.Unlocked {
		fmt.Printf(`Wallet status:
%v, Locked
Unlock the wallet to view balance
`, encStatus)
		return
	}

	unconfirmedBalance := status.ConfirmedCoinBalance.Add(status.UnconfirmedIncomingCoins).Sub(status.UnconfirmedOutgoingCoins)
	var delta string
	if unconfirmedBalance.Cmp(status.ConfirmedCoinBalance) >= 0 {
		delta = "+ " + _CurrencyConvertor.ToCoinStringWithUnit(unconfirmedBalance.Sub(status.ConfirmedCoinBalance))
	} else {
		delta = "- " + _CurrencyConvertor.ToCoinStringWithUnit(status.ConfirmedCoinBalance.Sub(unconfirmedBalance))
	}

	fmt.Printf(`Wallet status:
%s, Unlocked
Confirmed Balance:   %v
Locked Balance:      %v
Unconfirmed Delta:   %v
BlockStakes:         %v BS
`, encStatus, _CurrencyConvertor.ToCoinStringWithUnit(status.ConfirmedCoinBalance),
		_CurrencyConvertor.ToCoinStringWithUnit(status.ConfirmedLockedCoinBalance),
		delta, status.BlockStakeBalance)
	if !status.LockedBlockStakeBalance.IsZero() {
		fmt.Printf("Locked BlockStakes:  %v BS\n", status.LockedBlockStakeBalance)
	}
}

// wallettransactionscmd lists all of the transactions related to the wallet,
// providing a net flow of siacoins and siafunds for each.
func wallettransactionscmd() {
	wtg := new(api.WalletTransactionsGET)
	err := _DefaultClient.httpClient.GetAPI("/wallet/transactions?startheight=0&endheight=10000000", wtg)
	if err != nil {
		Die("Could not fetch transaction history:", err)
	}

	fmt.Println("    [height]                                                   [transaction id]       [net coins]   [net blockstakes]")
	txns := append(wtg.ConfirmedTransactions, wtg.UnconfirmedTransactions...)
	for _, txn := range txns {
		// Determine the number of outgoing siacoins and siafunds.
		var outgoingSiacoins types.Currency
		var outgoingBlockStakes types.Currency
		for _, input := range txn.Inputs {
			if input.FundType == types.SpecifierCoinInput && input.WalletAddress {
				outgoingSiacoins = outgoingSiacoins.Add(input.Value)
			}
			if input.FundType == types.SpecifierBlockStakeInput && input.WalletAddress {
				outgoingBlockStakes = outgoingBlockStakes.Add(input.Value)
			}
		}

		// Determine the number of incoming siacoins and siafunds.
		var incomingSiacoins types.Currency
		var incomingBlockStakes types.Currency
		for _, output := range txn.Outputs {
			if output.FundType == types.SpecifierMinerPayout {
				incomingSiacoins = incomingSiacoins.Add(output.Value)
			}
			if output.FundType == types.SpecifierCoinOutput && output.WalletAddress {
				incomingSiacoins = incomingSiacoins.Add(output.Value)
			}
			if output.FundType == types.SpecifierBlockStakeOutput && output.WalletAddress {
				incomingBlockStakes = incomingBlockStakes.Add(output.Value)
			}
		}

		// Convert the siacoins to a float.
		incomingSiacoinsFloat, _ := new(big.Rat).SetFrac(incomingSiacoins.Big(), _CurrencyUnits.OneCoin.Big()).Float64()
		outgoingSiacoinsFloat, _ := new(big.Rat).SetFrac(outgoingSiacoins.Big(), _CurrencyUnits.OneCoin.Big()).Float64()

		// Print the results.
		if txn.ConfirmationHeight < 1e9 {
			fmt.Printf("%12v", txn.ConfirmationHeight)
		} else {
			fmt.Printf(" unconfirmed")
		}
		fmt.Printf("%67v%15.2f", txn.TransactionID, incomingSiacoinsFloat-outgoingSiacoinsFloat)
		incomingBlockStakeBigInt := incomingBlockStakes.Big()
		outgoingBlockStakeBigInt := outgoingBlockStakes.Big()
		fmt.Printf("%14s BS\n", new(big.Int).Sub(incomingBlockStakeBigInt, outgoingBlockStakeBigInt).String())
	}
}

// walletunlockcmd unlocks a saved wallet
func walletunlockcmd() {
	password, err := speakeasy.Ask("Wallet password: ")
	if err != nil {
		Die("Reading password failed:", err)
	}
	fmt.Println("Unlocking the wallet. This may take several minutes...")
	qs := fmt.Sprintf("passphrase=%s", password)
	err = _DefaultClient.httpClient.Post("/wallet/unlock", qs)
	if err != nil {
		Die("Could not unlock wallet:", err)
	}
	fmt.Println("Wallet unlocked")
}

// walletsendtxncmd sends commits a transaction in json format
// to the transaction pool
func walletsendtxncmd(txnjson string) {
	var resp api.TransactionPoolPOST
	err := _DefaultClient.httpClient.PostResp("/transactionpool/transactions", txnjson, &resp)
	if err != nil {
		Die("Could not publish transaction:", err)
	}
	fmt.Println("Transaction published, transaction id:", resp.TransactionID)
}
