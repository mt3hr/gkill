package dvnf_cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

// pipeinしてからBOM除去、改行コード統一をして返します
func pipeInFormatted() (string, error) {
	b, err := pipeIn()
	if err != nil {
		err = fmt.Errorf("error at input from pipe: %w", err)
		return "", err
	}
	if b == nil {
		return "", nil
	}
	return formatPipein(b), nil
}

//stdinがterminalでなければ読み取ります？
//パイプから渡されたっぽければ、受け取ったデータを渡します。
func pipeIn() ([]byte, error) {
	//標準入力がパイプじゃなければ返す
	if stdinIsTerminal() {
		return nil, nil
	}

	//標準入力から読み取る
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		err = fmt.Errorf("error at read stdin: %w", err)
		return nil, err
	}

	return b, nil
}

//文字列から改行文字を除去します。
func removeNewlineCodes(str string) string {
	for _, nlcode := range newlineCodes {
		str = strings.Replace(str, nlcode, "", -1)
	}
	return str
}

// BOMを除去して、改行コードを\nに揃えます。
func formatPipein(pipein []byte) string {
	pipeStr := string(pipein)
	pipeStr = replaceNewlineCodes(pipeStr, "\n")
	pipeStr = removeBOM(pipeStr)
	return pipeStr
}

//標準入力がTerminalであればtrueを返します
func stdinIsTerminal() bool {
	fd := int(os.Stdin.Fd())
	return terminal.IsTerminal(fd)
}

//改行文字
var newlineCodes = []string{
	"\r\n",
	"\r",
	"\n",
}

//改行文字を\nに揃えます
func replaceNewlineCodes(str, new string) string {
	for _, nlcode := range newlineCodes {
		str = strings.Replace(str, nlcode, new, -1)
	}
	return str
}

//BOMのマップ
var boms = [][]byte{
	{0xEF, 0xBB, 0xBF},       //UTF-8
	{0xFE, 0xFF},             //UTF-16 BE
	{0xFF, 0xFE},             //UTF-16 LE
	{0x00, 0x00, 0xFE, 0xFF}, //UTF-32 BE
	{0xFF, 0xFE, 0x00, 0x00}, //UTF-32 LE
	{0x2B, 0x2F, 0x76, 0x38}, //UTF-7
	{0x2B, 0x2F, 0x76, 0x39}, //UTF-7
	{0x2B, 0x2F, 0x76, 0x2B}, //UTF-7
	{0x2B, 0x2F, 0x76, 0x2F}, //UTF-7
}

//頭にBOMがついていたら除去します。
func removeBOM(str string) string {
	b := []byte(str)
	for _, bom := range boms {
		if bytes.HasPrefix(b, bom) {
			b = bytes.TrimPrefix(b, bom)
			return string(b)
		}
	}
	return str
}
