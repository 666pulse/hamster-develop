package service

import (
	"fmt"
	"github.com/mohaijiang/agent-go"
	"github.com/mohaijiang/agent-go/candid"
	"github.com/mohaijiang/agent-go/identity"
	"github.com/mohaijiang/agent-go/principal"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestIcpService_CreateIdentity(t *testing.T) {
	useIdentityCmd := "dfx identity use test1113"
	_, err := execDfxCommand(useIdentityCmd)
	if err != nil {
		panic(err)
	}
	accountIdCmd := "dfx ledger account-id"
	accountId, err := execDfxCommand(accountIdCmd)
	if err != nil {
		panic(err)
	}
	pIdCmd := "dfx identity get-principal"
	pId, err := execDfxCommand(pIdCmd)
	if err != nil {
		panic(err)
	}
	fmt.Printf("identity: %s \naccount-id: %s \nprincipal-id: %s \n", "test1113", strings.TrimSpace(accountId), strings.TrimSpace(pId))
}

func TestIcpService_GetWalletIdByDfx(t *testing.T) {
	useIdentityCmd := "dfx identity use identity_abing"
	_, err := execDfxCommand(useIdentityCmd)
	if err != nil {
		panic(err)
	}
	getWalletCmd := "dfx identity get-wallet --network ic"
	output, err := execDfxCommand(getWalletCmd)
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(`([a-z0-9-]+-[a-z0-9-]+-[a-z0-9-]+-[a-z0-9-]+-[a-z0-9-]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		fmt.Printf("walletId is: %s \n", matches[1])
	} else {
		fmt.Errorf("fail to get walletId")
	}
}

func execDfxCommand(cmd string) (string, error) {
	output, err := exec.Command("bash", "-c", cmd).Output()
	if exitError, ok := err.(*exec.ExitError); ok {
		fmt.Errorf("%s Exit status: %d, Exit str: %s", cmd, exitError.ExitCode(), string(exitError.Stderr))
		return "", err
	} else if err != nil {
		// 输出其他类型的错误
		fmt.Errorf("%s Failed to execute command: %s", cmd, err)
		return "", err
	}
	return string(output), nil
}

func TestIcpService_InitWallet(t *testing.T) {
	str := `
Transfer sent at BlockHeight: 20
Canister created with id: "gastn-uqaaa-aaaae-aaafq-cai"
`

	// 定义正则表达式
	re := regexp.MustCompile(`Canister created with id: "(.*?)"`)

	// 使用正则表达式提取字符串
	matches := re.FindStringSubmatch(str)

	// 获取匹配到的字符串
	if len(matches) > 1 {
		value := matches[1]
		fmt.Println("Value:", value)
	} else {
		fmt.Println("String not found.")
	}
}

func TestIcpService_getLedgerIcpBalance(t *testing.T) {

	amount, err := strconv.ParseFloat("0.01010000", 64)
	if err != nil {
		panic(t)
	}
	fmt.Println(amount)
	if amount > 0.0002 {
		amount -= 0.0002
	} else {
		panic(t)
	}
	fmt.Println(amount)
	float := strconv.FormatFloat(amount, 'f', 8, 64)
	fmt.Println(float)
}

var ic0, _ = url.Parse("https://ic0.app/")

func TestIcpService_AgentGo(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Errorf("failed to get user home directory %s", err)
		panic(t)
	}

	// 构建文件路径
	filePath := filepath.Join(homeDir, ".config", "dfx", "identity", "test1115", "identity.pem")

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Errorf("failed to read file %s", err)
		panic(t)
	}
	fmt.Printf("dfx 生成的pem是\n  %s \n", string(content))
	//从pem文件获取新的标识
	ed25519Identity, _ := identity.NewRandomEd25519Identity()
	pem, _ := ed25519Identity.ToPEM()
	fmt.Printf("sdk 生成的pem是\n  %s \n", string(pem))
	id, err := identity.NewEd25519IdentityFromPEM(pem)

	//获取控制者
	c := agent.NewClient(agent.ClientConfig{Host: ic0})
	status, _ := c.Status()
	fmt.Println(status.Version)
	canisterId := "ryjl3-tyaaa-aaaaa-aaaba-cai"
	ledgerID, _ := principal.Decode(canisterId)
	a, err := agent.New(agent.Config{
		Identity:     id,
		ClientConfig: &agent.ClientConfig{Host: ic0},
	})
	if err != nil {
		panic(t)
	}
	//获取控制者
	controllers, err := a.GetCanisterControllers(ledgerID)
	if err != nil {
		fmt.Println(err)
		panic(t)
	}
	fmt.Printf("%s 的控制者是 %s \n", canisterId, controllers)

	//查询方法
	args, err := candid.EncodeValueString("record { account = \"9523dc824aa062dcd9c91b98f4594ff9c6af661ac96747daef2090b7fe87037d\" }")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(a.QueryString(ledgerID, "account_balance_dfx", args))

}
func TestIcpService_AgentGo_Fail(t *testing.T) {
	canisterId := "aaaaa-aa"
	ledgerID, _ := principal.Decode(canisterId)
	ed25519Identity, _ := identity.NewRandomEd25519Identity()
	a, err := agent.New(agent.Config{
		Identity:     ed25519Identity,
		ClientConfig: &agent.ClientConfig{Host: ic0},
	})
	if err != nil {
		panic(t)
	}
	//查询方法
	args, err := candid.EncodeValueString("record { canister_id = \"ryjl3-tyaaa-aaaaa-aaaba-cai\" }")
	if err != nil {
		fmt.Println(err)
		panic(t)
	}
	callRaw, err := a.QueryString(ledgerID, "canister_status", args)
	if err != nil {
		fmt.Println(err)
		panic(t)
	}
	fmt.Println(callRaw)
}
