// Just MVP for test idea, not a using for real production code. Official documentation how using this functional will be writted in docs/pyzero.md

package main

import (
	"encoding/json"
	"os"
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
		os.Stderr.WriteString("Error parsing python: " + err.Error() + "\n")
		return
	}

	var irInstructions []IRInstruction
	if err := json.Unmarshal(output, &irInstructions); err != nil {
		os.Stderr.WriteString("Error decoding IR: " + err.Error() + "\n")
		return
	}

	os.Stdout.WriteString("[ForgeZero] Received IR from Python frontend:\n")
	for _, inst := range irInstructions {
		os.Stdout.WriteString("Instruction: Op=" + inst.Op + ", Dest=" + inst.Dest + ", Arg1=" + inst.Arg1 + ", Arg2=" + inst.Arg2 + "\n")
	}
}