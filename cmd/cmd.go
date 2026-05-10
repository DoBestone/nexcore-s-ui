package cmd

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/alireza0/s-ui/cmd/migration"
	"github.com/alireza0/s-ui/config"
)

func ParseCmd() {
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "show version")

	adminCmd := flag.NewFlagSet("admin", flag.ExitOnError)
	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)
	tokenCmd := flag.NewFlagSet("token", flag.ExitOnError)
	reportCmd := flag.NewFlagSet("report", flag.ExitOnError)

	var username string
	var password string
	var port int
	var path string
	var reset bool
	var show bool
	settingCmd.BoolVar(&reset, "reset", false, "reset all settings")
	settingCmd.BoolVar(&show, "show", false, "show current settings")
	settingCmd.IntVar(&port, "port", 0, "set panel port")
	settingCmd.StringVar(&path, "path", "", "set panel path")

	adminCmd.BoolVar(&show, "show", false, "show first admin credentials")
	adminCmd.BoolVar(&reset, "reset", false, "reset first admin credentials")
	adminCmd.StringVar(&username, "username", "", "set login username")
	adminCmd.StringVar(&password, "password", "", "set login password")

	var tokenAdd bool
	var tokenList bool
	var tokenDel bool
	var tokenDesc string
	var tokenExpiry int64
	var tokenIdStr string
	tokenCmd.BoolVar(&tokenAdd, "add", false, "create a new admin scope token")
	tokenCmd.BoolVar(&tokenList, "list", false, "list all tokens (id desc prefix expiry)")
	tokenCmd.BoolVar(&tokenDel, "del", false, "delete token by id")
	tokenCmd.StringVar(&tokenDesc, "desc", "installer", "token description")
	tokenCmd.Int64Var(&tokenExpiry, "expiry", 0, "expiry days from now (0 = never expire)")
	tokenCmd.StringVar(&tokenIdStr, "id", "", "token id (for -del)")

	var reportURL string
	var reportAllowHTTP bool
	reportCmd.StringVar(&reportURL, "url", "", "webhook URL to POST install info to (HMAC-SHA256 signed)")
	reportCmd.BoolVar(&reportAllowHTTP, "allow-http", false, "allow plain http URL (testing only; default refuse)")

	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("    admin          set/reset/show first admin credentials")
		fmt.Println("    uri            Show panel URI")
		fmt.Println("    migrate        migrate form older version")
		fmt.Println("    setting        set/reset/show settings")
		fmt.Println("    token          create/list/delete admin scope API token")
		fmt.Println("    report         POST install info to webhook (HMAC-SHA256 signed)")
		fmt.Println()
		adminCmd.Usage()
		fmt.Println()
		settingCmd.Usage()
		fmt.Println()
		tokenCmd.Usage()
		fmt.Println()
		reportCmd.Usage()
	}

	flag.Parse()
	if showVersion {
		fmt.Println("nexcore-s-ui\t", config.GetVersion())
		info, ok := debug.ReadBuildInfo()
		if ok {
			for _, dep := range info.Deps {
				if dep.Path == "github.com/sagernet/sing-box" {
					fmt.Println("Sing-Box\t", dep.Version)
					break
				}
			}
		}
		return
	}

	switch os.Args[1] {
	case "admin":
		err := adminCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		switch {
		case show:
			showAdmin()
		case reset:
			resetAdmin()
		default:
			updateAdmin(username, password)
			showAdmin()
		}

	case "uri":
		getPanelURI()

	case "migrate":
		migration.MigrateDb()

	case "setting":
		err := settingCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		switch {
		case show:
			showSetting()
		case reset:
			resetSetting()
		default:
			updateSetting(port, path)
			showSetting()
		}

	case "token":
		err := tokenCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		switch {
		case tokenAdd:
			addToken(tokenDesc, tokenExpiry)
		case tokenList:
			listTokens()
		case tokenDel:
			if tokenIdStr == "" {
				fmt.Println("token -del requires -id")
				return
			}
			deleteToken(tokenIdStr)
		default:
			tokenCmd.Usage()
		}

	case "report":
		err := reportCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		os.Exit(runReport(reportURL, reportAllowHTTP))

	default:
		fmt.Println("Invalid subcommands")
		flag.Usage()
	}
}
