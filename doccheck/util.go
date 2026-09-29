package doccheck

import (
	"fmt"
	"strconv"
)

func sprintf(format string, args ...any) string { return fmt.Sprintf(format, args...) }

func itoa(n int) string { return strconv.Itoa(n) }
