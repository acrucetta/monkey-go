package compiler

import (
	"fmt"

	"github.com/kitasuke/monkey-go/ast"
	"github.com/kitasuke/monkey-go/code"
	"github.com/kitasuke/monkey-go/object"
)

type Compiler struct {
	instructions code.Instructions
	constants    []object.Object

	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

func New() *Compiler {
	return &Compiler{
		instructions:        code.Instructions{},
		constants:           []object.Object{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, statement := range node.Statements {
			if err := c.Compile(statement); err != nil {
				return err
			}
			c.emit(code.OpPop)
		}

	case *ast.BlockStatement:
		for _, statement := range node.Statements {
			if err := c.Compile(statement); err != nil {
				return err
			}
		}

	case *ast.IfExpression:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		// emit an `OpJumpNotTruthy` with a bogus value
		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		if c.lastInstructionIsPop() {
			c.removeLastPop()
		}

		afterConsequencePos := len(c.instructions)
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		if node.Alternative == nil {
			afterConsequencePos := len(c.instructions)
			c.changeOperand(jumpNotTruthyPos, afterConsequencePos)
		} else {
			// emit an `OpJump` with a bogus value
			jumpPos := c.emit(code.OpJump, 9999)

			afterConsequencePos := len(c.instructions)
			c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

			err = c.Compile(node.Alternative)
			if err != nil {
				return err
			}

			if c.lastInstructionIsPop() {
				c.removeLastPop()
			}

			afterAlternativePos := len(c.instructions)
			c.changeOperand(jumpPos, afterAlternativePos)
		}

	case *ast.ExpressionStatement:
		if err := c.Compile(node.Expression); err != nil {
			return err
		}

	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}

	case *ast.InfixExpression:

		if node.Operator == "<" {
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}
			err = c.Compile(node.Left)
			if err != nil {
				return err
			}
			c.emit(code.OpGreaterThan)
			return nil
		}

		if err := c.Compile(node.Left); err != nil {
			return err
		}

		if err := c.Compile(node.Right); err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		case ">":
			c.emit(code.OpGreaterThan)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))
	}

	return nil
}

func (c *Compiler) lastInstructionIsPop() bool {
	return c.lastInstruction.Opcode == code.OpPop
}

func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	for i := 0; i < len(newInstruction); i++ {
		c.instructions[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.instructions[opPos])
	newInstruction := code.Make(op, operand)

	c.replaceInstruction(opPos, newInstruction)
}

func (c *Compiler) removeLastPop() {
	c.instructions = c.instructions[:len(c.instructions)-1]
	c.lastInstruction = c.previousInstruction
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	instruction := code.Make(op, operands...)
	position := c.addInstruction(instruction)

	c.setLastInstruction(op, position)

	return position
}

func (c *Compiler) setLastInstruction(op code.Opcode, position int) {
	previous := c.lastInstruction
	last := EmittedInstruction{Opcode: op, Position: position}

	c.previousInstruction = previous
	c.lastInstruction = last
}

func (c *Compiler) addInstruction(instruction []byte) int {
	positionNewInstruction := len(c.instructions)
	c.instructions = append(c.instructions, instruction...)
	return positionNewInstruction
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
	}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}
