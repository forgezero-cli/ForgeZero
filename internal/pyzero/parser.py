'''
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
'''



import ast
import json
import sys


class IRGenerator(ast.NodeVisitor):
    def __init__(self):
        self.instructions = []
        self.temp_counter = 0

    def new_temp(self):
        self.temp_counter += 1
        return f"%t{self.temp_counter}"

    def visit_Assign(self, node):
        target = node.targets[0].id

        if isinstance(node.value, ast.BinOp):
            left = self.visit(node.value.left)
            right = self.visit(node.value.right)
            op_type = type(node.value.op).__name__

            temp = self.new_temp()
            self.instructions.append(
                {"op": op_type, "dest": temp, "arg1": left, "arg2": right}
            )
            self.instructions.append({"op": "Store", "dest": target, "arg1": temp})
        elif isinstance(node.value, ast.Constant):
            self.instructions.append(
                {"op": "Store", "dest": target, "arg1": str(node.value.value)}
            )

    def visit_Name(self, node):
        return node.id

    def visit_Constant(self, node):
        return str(node.value)

    def visit_Call(self, node):
        if isinstance(node.func, ast.Name) and node.func.id == "print":
            arg = self.visit(node.args[0])
            self.instructions.append({"op": "Print", "arg1": arg})


if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit(1)

    with open(sys.argv[1], "r") as f:
        code = f.read()

    tree = ast.parse(code)
    generator = IRGenerator()
    generator.visit(tree)

    print(json.dumps(generator.instructions, indent=2))

