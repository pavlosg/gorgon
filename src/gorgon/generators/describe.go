package generators

import (
	"fmt"
	"strconv"

	"github.com/pavlosg/gorgon/src/gorgon"
)

func DescribeOperation(input gorgon.Instruction, output interface{}) string {
	returnValue := "?"
	if output == nil {
		returnValue = "nil"
	} else {
		switch rv := output.(type) {
		case int:
			returnValue = strconv.Itoa(rv)
		case string:
			returnValue = rv
		case interface{ String() string }:
			returnValue = rv.String()
		case error:
			str := rv.Error()
			if len(str) > 30 {
				str = str[:30] + "…"
			}
			returnValue = fmt.Sprintf("error:%q", str)
		}
	}
	return input.String() + " → " + returnValue
}
