// Just MVP for test idea, not a using for real production code. Official documentation how using this functional will be writted in docs/pyzero.md

package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type IRInstruction struct {
	Op   string `json:"op"`
	Dest string `json:"dest"`
	Arg1 string `json:"arg1"`
	Arg2 string `json:"arg2"`
}

func main() {
	cmd := exec.Command("python3", "parser.py", "test.py")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error parsing python: %v\n", err)
		return
	}

	var irInstructions []IRInstruction
	if err := json.Unmarshal(output, &irInstructions); err != nil {
		fmt.Printf("Error decoding IR: %v\n", err)
		return
	}

	fmt.Println("[ForgeZero] Received IR from Python frontend:")
	for _, inst := range irInstructions {
		fmt.Printf("Instruction: Op=%s, Dest=%s, Arg1=%s, Arg2=%s\n", inst.Op, inst.Dest, inst.Arg1, inst.Arg2)
	}
}
