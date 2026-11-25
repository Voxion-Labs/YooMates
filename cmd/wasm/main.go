//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"yoomates/internal/search"
)

func searchWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 2 {
			return `{"query":"","total":0,"results":[]}`
		}
		query := args[0].String()
		messagesJSON := args[1].String()

		res, err := search.ExecuteSearchJSON(query, messagesJSON)
		if err != nil {
			return fmt.Sprintf(`{"error":%q}`, err.Error())
		}
		return res
	})
}

func main() {
	js.Global().Set("YooSearchWASM", searchWrapper())
	js.Global().Set("YooSearchReady", js.ValueOf(true))

	// Keep the WebAssembly runtime active
	select {}
}
