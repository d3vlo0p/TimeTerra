package controller

import "fmt"

func ResourceName(kind string, name string) string {
	return fmt.Sprintf("%s:%s", kind, name)
}
