package js

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/tdewolff/parse/v2"
)

var ErrInvalidJSON = fmt.Errorf("invalid JSON")

type JSONer interface {
	JSON(*bytes.Buffer) error
}

// AST is the full ECMAScript abstract syntax tree.
type AST struct {
	BlockStmt // module
}

func (ast *AST) String() string {
	s := ""
	for i, item := range ast.BlockStmt.List {
		if i != 0 {
			s += " "
		}
		s += item.String()
	}
	return s
}

////////////////////////////////////////////////////////////////

// DeclType specifies the kind of declaration.
type DeclType uint16

// DeclType values.
const (
	NoDecl       DeclType = iota // undeclared variables
	VariableDecl                 // var
	FunctionDecl                 // function
	ArgumentDecl                 // function and method arguments
	LexicalDecl                  // let, const, class
	CatchDecl                    // catch statement argument
	ExprDecl                     // function expression name or class expression name
)

func (decl DeclType) String() string {
	switch decl {
	case NoDecl:
		return "NoDecl"
	case VariableDecl:
		return "VariableDecl"
	case FunctionDecl:
		return "FunctionDecl"
	case ArgumentDecl:
		return "ArgumentDecl"
	case LexicalDecl:
		return "LexicalDecl"
	case CatchDecl:
		return "CatchDecl"
	case ExprDecl:
		return "ExprDecl"
	}
	return "Invalid(" + strconv.Itoa(int(decl)) + ")"
}

// Var is a variable, where Decl is the type of declaration and can be var|function for function scoped variables, let|const|class for block scoped variables.
type Var struct {
	Data []byte
	Link *Var // is set when merging variable uses, as in:  {a} {var a}  where the first links to the second, only used for undeclared variables
	Uses uint16
	Decl DeclType
}

// Name returns the variable name.
func (v *Var) Name() []byte {
	for v.Link != nil {
		v = v.Link
	}
	return v.Data
}

func (v Var) String() string {
	return string(v.Name())
}

// JS converts the node back to valid JavaScript
func (v Var) JS() string {
	return v.String()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (v Var) JSWriteTo(w io.Writer) (i int, err error) {
	return w.Write(v.Name())
}

// VarsByUses is sortable by uses in descending order.
// TODO: write custom sorter for varsbyuses
type VarsByUses VarArray

func (vs VarsByUses) Len() int {
	return len(vs)
}

func (vs VarsByUses) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs VarsByUses) Less(i, j int) bool {
	return vs[i].Uses > vs[j].Uses
}

////////////////////////////////////////////////////////////////

// VarArray is a set of variables in scopes.
type VarArray []*Var

func (vs VarArray) String() string {
	s := "["
	for i, v := range vs {
		if i != 0 {
			s += ", "
		}
		links := 0
		for v.Link != nil {
			v = v.Link
			links++
		}
		s += fmt.Sprintf("Var{%v %s %v %v}", v.Decl, string(v.Data), links, v.Uses)
	}
	return s + "]"
}

// Scope is a function or block scope with a list of variables declared and used.
type Scope struct {
	Parent, Func   *Scope   // Parent is nil for global scope
	Declared       VarArray // Link in Var are always nil
	Undeclared     VarArray
	VarDecls       []*VarDecl
	NumForDecls    uint16 // offset into Declared to mark variables used in for statements
	NumFuncArgs    uint16 // offset into Declared to mark variables used in function arguments
	NumArgUses     uint16 // offset into Undeclared to mark variables used in arguments
	IsGlobalOrFunc bool
	HasWith        bool
}

func (s Scope) String() string {
	return "Scope{Declared: " + s.Declared.String() + ", Undeclared: " + s.Undeclared.String() + "}"
}

// Declare declares a new variable.
func (s *Scope) Declare(decl DeclType, name []byte) (*Var, bool) {
	// refer to new variable for previously undeclared symbols in the current and lower scopes
	// this happens in `{ a = 5; } var a` where both a's refer to the same variable
	curScope := s
	if decl == VariableDecl || decl == FunctionDecl {
		// find function scope for var and function declarations
		for s != s.Func {
			// make sure that `{let i;{var i}}` is an error
			if v := s.findDeclared(name, false); v != nil && v.Decl != decl && v.Decl != CatchDecl {
				return nil, false
			}
			s = s.Parent
		}
	}

	if v := s.findDeclared(name, true); v != nil {
		// variable already declared, might be an error or a duplicate declaration
		if (ArgumentDecl < v.Decl || FunctionDecl < decl) && v.Decl != ExprDecl {
			// only allow (v.Decl,decl) of: (var|function|argument,var|function), (expr,*), any other combination is a syntax error
			return nil, false
		}
		if v.Decl == ExprDecl {
			v.Decl = decl
		}
		v.Uses++
		for s != curScope {
			curScope.AddUndeclared(v) // add variable declaration as used variable to the current scope
			curScope = curScope.Parent
		}
		return v, true
	}

	var v *Var
	// reuse variable if previously used, as in:  a;var a
	if decl != ArgumentDecl { // in case of function f(a=b,b), where the first b is different from the second
		for i, uv := range s.Undeclared[s.NumArgUses:] {
			// no need to evaluate v.Link as v.Data stays the same and Link is nil in the active scope
			if 0 < uv.Uses && uv.Decl == NoDecl && bytes.Equal(name, uv.Data) {
				// must be NoDecl so that it can't be a var declaration that has been added
				v = uv
				s.Undeclared = append(s.Undeclared[:int(s.NumArgUses)+i], s.Undeclared[int(s.NumArgUses)+i+1:]...)
				break
			}
		}
	}
	if v == nil {
		// add variable to the context list and to the scope
		v = &Var{name, nil, 0, decl}
	} else {
		v.Decl = decl
	}
	v.Uses++
	s.Declared = append(s.Declared, v)
	for s != curScope {
		curScope.AddUndeclared(v) // add variable declaration as used variable to the current scope
		curScope = curScope.Parent
	}
	return v, true
}

// Use increments the usage of a variable.
func (s *Scope) Use(name []byte) *Var {
	// check if variable is declared in the current scope
	v := s.findDeclared(name, false)
	if v == nil {
		// check if variable is already used before in the current or lower scopes
		v = s.findUndeclared(name)
		if v == nil {
			// add variable to the context list and to the scope's undeclared
			v = &Var{name, nil, 0, NoDecl}
			s.Undeclared = append(s.Undeclared, v)
		}
	}
	v.Uses++
	return v
}

// findDeclared finds a declared variable in the current scope.
func (s *Scope) findDeclared(name []byte, skipForDeclared bool) *Var {
	start := 0
	if skipForDeclared {
		// we skip the for initializer for declarations (only has effect for let/const)
		start = int(s.NumForDecls)
	}
	// reverse order to find the inner let first in `for(let a in []){let a; {a}}`
	for i := len(s.Declared) - 1; start <= i; i-- {
		v := s.Declared[i]
		// no need to evaluate v.Link as v.Data stays the same, and Link is always nil in Declared
		if bytes.Equal(name, v.Data) {
			return v
		}
	}
	return nil
}

// findUndeclared finds an undeclared variable in the current and contained scopes.
func (s *Scope) findUndeclared(name []byte) *Var {
	for _, v := range s.Undeclared {
		// no need to evaluate v.Link as v.Data stays the same and Link is nil in the active scope
		if 0 < v.Uses && bytes.Equal(name, v.Data) {
			return v
		}
	}
	return nil
}

// add undeclared variable to scope, this is called for the block scope when declaring a var in it
func (s *Scope) AddUndeclared(v *Var) {
	// don't add undeclared symbol if it's already there
	for _, vorig := range s.Undeclared {
		if v == vorig {
			return
		}
	}
	s.Undeclared = append(s.Undeclared, v) // add variable declaration as used variable to the current scope
}

// MarkForStmt marks the declared variables in current scope as for statement initializer to distinguish from declarations in body.
func (s *Scope) MarkForStmt() {
	s.NumForDecls = uint16(len(s.Declared))
	s.NumArgUses = uint16(len(s.Undeclared)) // ensures for different b's in for(var a in b){let b}
}

// MarkFuncArgs marks the declared/undeclared variables in the current scope as function arguments.
func (s *Scope) MarkFuncArgs() {
	s.NumFuncArgs = uint16(len(s.Declared))
	s.NumArgUses = uint16(len(s.Undeclared)) // ensures different b's in `function f(a=b){var b}`.
}

// HoistUndeclared copies all undeclared variables of the current scope to the parent scope.
func (s *Scope) HoistUndeclared() {
	for i, vorig := range s.Undeclared {
		// no need to evaluate vorig.Link as vorig.Data stays the same
		if 0 < vorig.Uses && vorig.Decl == NoDecl {
			if v := s.Parent.findDeclared(vorig.Data, false); v != nil {
				// check if variable is declared in parent scope
				v.Uses += vorig.Uses
				vorig.Link = v
				s.Undeclared[i] = v // point reference to existing var (to avoid many Link chains)
			} else if v := s.Parent.findUndeclared(vorig.Data); v != nil {
				// check if variable is already used before in parent scope
				v.Uses += vorig.Uses
				vorig.Link = v
				s.Undeclared[i] = v // point reference to existing var (to avoid many Link chains)
			} else {
				// add variable to the context list and to the scope's undeclared
				s.Parent.Undeclared = append(s.Parent.Undeclared, vorig)
			}
		}
	}
}

// UndeclareScope undeclares all declared variables in the current scope and adds them to the parent scope.
// Called when possible arrow func ends up being a parenthesized expression, scope is not further used.
func (s *Scope) UndeclareScope() {
	// look if the variable already exists in the parent scope, if so replace the Var pointer in original use
	for _, vorig := range s.Declared {
		// no need to evaluate vorig.Link as vorig.Data stays the same, and Link is always nil in Declared
		// vorig.Uses will be atleast 1
		if v := s.Parent.findDeclared(vorig.Data, false); v != nil {
			// check if variable has been declared in this scope
			v.Uses += vorig.Uses
			vorig.Link = v
		} else if v := s.Parent.findUndeclared(vorig.Data); v != nil {
			// check if variable is already used before in the current or lower scopes
			v.Uses += vorig.Uses
			vorig.Link = v
		} else {
			// add variable to the context list and to the scope's undeclared
			vorig.Decl = NoDecl
			s.Parent.Undeclared = append(s.Parent.Undeclared, vorig)
		}
	}
	s.Declared = s.Declared[:0]
	s.Undeclared = s.Undeclared[:0]
}

// Unscope moves all declared variables of the current scope to the parent scope. Undeclared variables are already in the parent scope.
func (s *Scope) Unscope() {
	for _, vorig := range s.Declared {
		// no need to evaluate vorig.Link as vorig.Data stays the same, and Link is always nil in Declared
		// vorig.Uses will be atleast 1
		s.Parent.Declared = append(s.Parent.Declared, vorig)
	}
	s.Declared = s.Declared[:0]
	s.Undeclared = s.Undeclared[:0]
}

////////////////////////////////////////////////////////////////

// INode is an interface for AST nodes
type INode interface {
	String() string
	JS() string
	JSWriteTo(io.Writer) (int, error)
}

// IStmt is a dummy interface for statements.
type IStmt interface {
	INode
	stmtNode()
}

// IBinding is a dummy interface for bindings.
type IBinding interface {
	INode
	bindingNode()
}

// IExpr is a dummy interface for expressions.
type IExpr interface {
	INode
	exprNode()
}

////////////////////////////////////////////////////////////////

// Comment block or line, usually a bang comment.
type Comment struct {
	Value []byte
}

func (n Comment) String() string {
	return "Stmt(" + string(n.Value) + ")"
}

// JS converts the node back to valid JavaScript
func (n Comment) JS() string {
	return string(n.Value)
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n Comment) JSWriteTo(w io.Writer) (i int, err error) {
	return w.Write(n.Value)
}

// BlockStmt is a block statement.
type BlockStmt struct {
	List []IStmt
	Scope
}

func (n BlockStmt) String() string {
	s := "Stmt({"
	for _, item := range n.List {
		s += " " + item.String()
	}
	return s + " })"
}

// JS converts the node back to valid JavaScript
func (n BlockStmt) JS() string {
	s := ""
	if n.Scope.Parent != nil {
		s += "{ "
	}
	for _, item := range n.List {
		if _, isEmpty := item.(*EmptyStmt); !isEmpty {
			s += item.JS() + "; "
		}
	}
	if n.Scope.Parent != nil {
		s += "}"
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n BlockStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Scope.Parent != nil {
		wn, err = w.Write([]byte("{ "))
		i += wn
		if err != nil {
			return
		}
	}
	for _, item := range n.List {
		if _, isEmpty := item.(*EmptyStmt); !isEmpty {
			wn, err = item.JSWriteTo(w)
			i += wn
			if err != nil {
				return
			}
			wn, err = w.Write([]byte("; "))
			i += wn
			if err != nil {
				return
			}
		}
	}
	if n.Scope.Parent != nil {
		wn, err = w.Write([]byte{'}'})
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// EmptyStmt is an empty statement.
type EmptyStmt struct {
}

func (n EmptyStmt) String() string {
	return "Stmt(;)"
}

// JS converts the node back to valid JavaScript
func (n EmptyStmt) JS() string {
	return ";"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n EmptyStmt) JSWriteTo(w io.Writer) (i int, err error) {
	wn, err := w.Write([]byte{';'})
	i = wn
	return
}

// ExprStmt is an expression statement.
type ExprStmt struct {
	Value IExpr
}

func (n ExprStmt) String() string {
	val := n.Value.String()
	if val[0] == '(' && val[len(val)-1] == ')' {
		return "Stmt" + n.Value.String()
	}
	return "Stmt(" + n.Value.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n ExprStmt) JS() string {
	return n.Value.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ExprStmt) JSWriteTo(w io.Writer) (i int, err error) {
	return n.Value.JSWriteTo(w)
}

// IfStmt is an if statement.
type IfStmt struct {
	Cond IExpr
	Body IStmt
	Else IStmt // can be nil
}

func (n IfStmt) String() string {
	s := "Stmt(if " + n.Cond.String() + " " + n.Body.String()
	if n.Else != nil {
		s += " else " + n.Else.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n IfStmt) JS() string {
	s := "if (" + n.Cond.JS() + ") "
	switch n.Body.(type) {
	case *BlockStmt:
		s += n.Body.JS()
	default:
		s += "{ " + n.Body.JS() + " }"
	}
	if n.Else != nil {
		switch n.Else.(type) {
		case *BlockStmt:
			s += " else " + n.Else.JS()
		default:
			s += " else { " + n.Else.JS() + " }"
		}
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n IfStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("if ("))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Cond.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(") "))
	i += wn
	if err != nil {
		return
	}
	switch n.Body.(type) {
	case *BlockStmt:
		wn, err = n.Body.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	default:
		wn, err = w.Write([]byte("{ "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Body.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write([]byte(" }"))
		i += wn
		if err != nil {
			return
		}
	}
	if n.Else != nil {
		switch n.Else.(type) {
		case *BlockStmt:
			wn, err = w.Write([]byte(" else "))
			i += wn
			if err != nil {
				return
			}
			wn, err = n.Else.JSWriteTo(w)
			i += wn
			if err != nil {
				return
			}
		default:
			wn, err = w.Write([]byte(" else { "))
			i += wn
			if err != nil {
				return
			}
			wn, err = n.Else.JSWriteTo(w)
			i += wn
			if err != nil {
				return
			}
			wn, err = w.Write([]byte(" }"))
			i += wn
			if err != nil {
				return
			}
		}
	}
	return
}

// DoWhileStmt is a do-while iteration statement.
type DoWhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n DoWhileStmt) String() string {
	return "Stmt(do " + n.Body.String() + " while " + n.Cond.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n DoWhileStmt) JS() string {
	s := "do "
	switch n.Body.(type) {
	case *BlockStmt:
		s += n.Body.JS()
	default:
		s += "{ " + n.Body.JS() + " }"
	}
	return s + " while (" + n.Cond.JS() + ")"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n DoWhileStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("do "))
	i += wn
	if err != nil {
		return
	}
	switch n.Body.(type) {
	case *BlockStmt:
		wn, err = n.Body.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	default:
		wn, err = w.Write([]byte("{ "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Body.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write([]byte(" }"))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte(" while ("))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Cond.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(")"))
	i += wn
	return
}

// WhileStmt is a while iteration statement.
type WhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WhileStmt) String() string {
	return "Stmt(while " + n.Cond.String() + " " + n.Body.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n WhileStmt) JS() string {
	s := "while (" + n.Cond.JS() + ") "
	if n.Body != nil {
		s += n.Body.JS()
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n WhileStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("while ("))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Cond.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(") "))
	i += wn
	if err != nil {
		return
	}
	if n.Body != nil {
		wn, err = n.Body.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// ForStmt is a regular for iteration statement.
type ForStmt struct {
	Init IExpr // can be nil
	Cond IExpr // can be nil
	Post IExpr // can be nil
	Body *BlockStmt
}

func (n ForStmt) String() string {
	s := "Stmt(for"
	if v, ok := n.Init.(*VarDecl); !ok && n.Init != nil || ok && len(v.List) != 0 {
		s += " " + n.Init.String()
	}
	s += " ;"
	if n.Cond != nil {
		s += " " + n.Cond.String()
	}
	s += " ;"
	if n.Post != nil {
		s += " " + n.Post.String()
	}
	return s + " " + n.Body.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n ForStmt) JS() string {
	s := "for ("
	if v, ok := n.Init.(*VarDecl); !ok && n.Init != nil || ok && len(v.List) != 0 {
		s += n.Init.JS()
	} else {
		s += " "
	}
	s += "; "
	if n.Cond != nil {
		s += n.Cond.JS()
	}
	s += "; "
	if n.Post != nil {
		s += n.Post.JS()
	}
	return s + ") " + n.Body.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ForStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("for ("))
	i += wn
	if err != nil {
		return
	}
	if v, ok := n.Init.(*VarDecl); !ok && n.Init != nil || ok && len(v.List) != 0 {
		wn, err = n.Init.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	} else {
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte("; "))
	i += wn
	if err != nil {
		return
	}
	if n.Cond != nil {
		wn, err = n.Cond.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte("; "))
	i += wn
	if err != nil {
		return
	}
	if n.Post != nil {
		wn, err = n.Post.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte(") "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Body.JSWriteTo(w)
	i += wn
	return
}

// ForInStmt is a for-in iteration statement.
type ForInStmt struct {
	Init  IExpr
	Value IExpr
	Body  *BlockStmt
}

func (n ForInStmt) String() string {
	return "Stmt(for " + n.Init.String() + " in " + n.Value.String() + " " + n.Body.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n ForInStmt) JS() string {
	return "for (" + n.Init.JS() + " in " + n.Value.JS() + ") " + n.Body.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ForInStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("for ("))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Init.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(" in "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Value.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(") "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Body.JSWriteTo(w)
	i += wn
	return
}

// ForOfStmt is a for-of iteration statement.
type ForOfStmt struct {
	Await bool
	Init  IExpr
	Value IExpr
	Body  *BlockStmt
}

func (n ForOfStmt) String() string {
	s := "Stmt(for"
	if n.Await {
		s += " await"
	}
	return s + " " + n.Init.String() + " of " + n.Value.String() + " " + n.Body.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n ForOfStmt) JS() string {
	s := "for"
	if n.Await {
		s += " await"
	}
	return s + " (" + n.Init.JS() + " of " + n.Value.JS() + ") " + n.Body.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ForOfStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("for"))
	i += wn
	if err != nil {
		return
	}
	if n.Await {
		wn, err = w.Write([]byte(" await"))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte(" ("))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Init.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(" of "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Value.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(") "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Body.JSWriteTo(w)
	i += wn
	return
}

// CaseClause is a case clause or default clause for a switch statement.
type CaseClause struct {
	TokenType
	Cond IExpr // can be nil
	List []IStmt
}

func (n CaseClause) String() string {
	s := " Clause(" + n.TokenType.String()
	if n.Cond != nil {
		s += " " + n.Cond.String()
	}
	for _, item := range n.List {
		s += " " + item.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n CaseClause) JS() string {
	s := " "
	if n.Cond != nil {
		s += "case " + n.Cond.JS()
	} else {
		s += "default"
	}
	s += ":"
	for _, item := range n.List {
		s += " " + item.JS() + ";"
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n CaseClause) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte(" "))
	i += wn
	if err != nil {
		return
	}
	if n.Cond != nil {
		wn, err = w.Write([]byte("case "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Cond.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	} else {
		wn, err = w.Write([]byte("default"))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte(":"))
	i += wn
	if err != nil {
		return
	}
	for _, item := range n.List {
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = item.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write([]byte(";"))
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// SwitchStmt is a switch statement.
type SwitchStmt struct {
	Init IExpr
	List []CaseClause
	Scope
}

func (n SwitchStmt) String() string {
	s := "Stmt(switch " + n.Init.String()
	for _, clause := range n.List {
		s += clause.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n SwitchStmt) JS() string {
	s := "switch (" + n.Init.JS() + ") {"
	for _, clause := range n.List {
		s += clause.JS()
	}
	return s + " }"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n SwitchStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("switch ("))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Init.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(") {"))
	i += wn
	if err != nil {
		return
	}
	for _, clause := range n.List {
		wn, err = clause.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte(" }"))
	i += wn
	return
}

// BranchStmt is a continue or break statement.
type BranchStmt struct {
	Type  TokenType
	Label []byte // can be nil
}

func (n BranchStmt) String() string {
	s := "Stmt(" + n.Type.String()
	if n.Label != nil {
		s += " " + string(n.Label)
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n BranchStmt) JS() string {
	s := n.Type.String()
	if n.Label != nil {
		s += " " + string(n.Label)
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n BranchStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write(n.Type.Bytes())
	i += wn
	if err != nil {
		return
	}
	if n.Label != nil {
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write(n.Label)
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// ReturnStmt is a return statement.
type ReturnStmt struct {
	Value IExpr // can be nil
}

func (n ReturnStmt) String() string {
	s := "Stmt(return"
	if n.Value != nil {
		s += " " + n.Value.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n ReturnStmt) JS() string {
	s := "return"
	if n.Value != nil {
		s += " " + n.Value.JS()
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ReturnStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("return"))
	i += wn
	if err != nil {
		return
	}
	if n.Value != nil {
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Value.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// WithStmt is a with statement.
type WithStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WithStmt) String() string {
	return "Stmt(with " + n.Cond.String() + " " + n.Body.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n WithStmt) JS() string {
	return "with (" + n.Cond.JS() + ") " + n.Body.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n WithStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("with ("))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Cond.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(") "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Body.JSWriteTo(w)
	i += wn
	return
}

// LabelledStmt is a labelled statement.
type LabelledStmt struct {
	Label []byte
	Value IStmt
}

func (n LabelledStmt) String() string {
	return "Stmt(" + string(n.Label) + " : " + n.Value.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n LabelledStmt) JS() string {
	return string(n.Label) + ": " + n.Value.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n LabelledStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write(n.Label)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(": "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Value.JSWriteTo(w)
	i += wn
	return
}

// ThrowStmt is a throw statement.
type ThrowStmt struct {
	Value IExpr
}

func (n ThrowStmt) String() string {
	return "Stmt(throw " + n.Value.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n ThrowStmt) JS() string {
	return "throw " + n.Value.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ThrowStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("throw "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Value.JSWriteTo(w)
	i += wn
	return
}

// TryStmt is a try statement.
type TryStmt struct {
	Body    *BlockStmt
	Binding IBinding   // can be nil
	Catch   *BlockStmt // can be nil
	Finally *BlockStmt // can be nil
}

func (n TryStmt) String() string {
	s := "Stmt(try " + n.Body.String()
	if n.Catch != nil {
		s += " catch"
		if n.Binding != nil {
			s += " Binding(" + n.Binding.String() + ")"
		}
		s += " " + n.Catch.String()
	}
	if n.Finally != nil {
		s += " finally " + n.Finally.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n TryStmt) JS() string {
	s := "try " + n.Body.JS()
	if n.Catch != nil {
		s += " catch"
		if n.Binding != nil {
			s += "(" + n.Binding.JS() + ")"
		}
		s += " " + n.Catch.JS()
	}
	if n.Finally != nil {
		s += " finally " + n.Finally.JS()
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n TryStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("try "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Body.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	if n.Catch != nil {
		wn, err = w.Write([]byte(" catch"))
		i += wn
		if err != nil {
			return
		}
		if n.Binding != nil {
			wn, err = w.Write([]byte("("))
			i += wn
			if err != nil {
				return
			}
			wn, err = n.Binding.JSWriteTo(w)
			i += wn
			if err != nil {
				return
			}
			wn, err = w.Write([]byte(")"))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Catch.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	if n.Finally != nil {
		wn, err = w.Write([]byte(" finally "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Finally.JSWriteTo(w)
		i += wn
	}
	return
}

// DebuggerStmt is a debugger statement.
type DebuggerStmt struct {
}

func (n DebuggerStmt) String() string {
	return "Stmt(debugger)"
}

// JS converts the node back to valid JavaScript
func (n DebuggerStmt) JS() string {
	return "debugger"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n DebuggerStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("debugger"))
	i += wn
	return
}

// Alias is a name space import or import/export specifier for import/export statements.
type Alias struct {
	Name    []byte // can be nil
	Binding []byte // can be nil
}

func (alias Alias) String() string {
	s := ""
	if alias.Name != nil {
		s += string(alias.Name) + " as "
	}
	return s + string(alias.Binding)
}

// JS converts the node back to valid JavaScript
func (alias Alias) JS() string {
	return alias.String()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (alias Alias) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if alias.Name != nil {
		wn, err = w.Write(alias.Name)
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write([]byte(" as "))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write(alias.Binding)
	i += wn
	return
}

// ImportStmt is an import statement.
type ImportStmt struct {
	List    []Alias
	Default []byte // can be nil
	Module  []byte
}

func (n ImportStmt) String() string {
	s := "Stmt(import"
	if n.Default != nil {
		s += " " + string(n.Default)
		if len(n.List) != 0 {
			s += " ,"
		}
	}
	if len(n.List) == 1 && len(n.List[0].Name) == 1 && n.List[0].Name[0] == '*' {
		s += " " + n.List[0].String()
	} else if 0 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String()
			}
		}
		s += " }"
	}
	if n.Default != nil || len(n.List) != 0 {
		s += " from"
	}
	return s + " " + string(n.Module) + ")"
}

// JS converts the node back to valid JavaScript
func (n ImportStmt) JS() string {
	s := "import"
	if n.Default != nil {
		s += " " + string(n.Default)
		if len(n.List) != 0 {
			s += " ,"
		}
	}
	if len(n.List) == 1 && len(n.List[0].Name) == 1 && n.List[0].Name[0] == '*' {
		s += " " + n.List[0].JS()
	} else if 0 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.JS()
			}
		}
		s += " }"
	}
	if n.Default != nil || len(n.List) != 0 {
		s += " from"
	}
	return s + " " + string(n.Module)
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ImportStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("import"))
	i += wn
	if err != nil {
		return
	}
	if n.Default != nil {
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write(n.Default)
		i += wn
		if err != nil {
			return
		}
		if len(n.List) != 0 {
			wn, err = w.Write([]byte(" ,"))
			i += wn
			if err != nil {
				return
			}
		}
	}
	if len(n.List) == 1 && len(n.List[0].Name) == 1 && n.List[0].Name[0] == '*' {
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.List[0].JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	} else if 0 < len(n.List) {
		wn, err = w.Write([]byte(" {"))
		i += wn
		if err != nil {
			return
		}
		for j, item := range n.List {
			if j != 0 {
				wn, err = w.Write([]byte(" ,"))
				i += wn
				if err != nil {
					return
				}
			}
			if item.Binding != nil {
				wn, err = w.Write([]byte(" "))
				i += wn
				if err != nil {
					return
				}
				wn, err = item.JSWriteTo(w)
				i += wn
				if err != nil {
					return
				}
			}
		}
		wn, err = w.Write([]byte(" }"))
		i += wn
		if err != nil {
			return
		}
	}
	if n.Default != nil || len(n.List) != 0 {
		wn, err = w.Write([]byte(" from"))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte(" "))
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write(n.Module)
	i += wn
	return
}

// ExportStmt is an export statement.
type ExportStmt struct {
	List    []Alias
	Module  []byte // can be nil
	Default bool
	Decl    IExpr
}

func (n ExportStmt) String() string {
	s := "Stmt(export"
	if n.Decl != nil {
		if n.Default {
			s += " default"
		}
		return s + " " + n.Decl.String() + ")"
	} else if len(n.List) == 1 && (len(n.List[0].Name) == 1 && n.List[0].Name[0] == '*' || n.List[0].Name == nil && len(n.List[0].Binding) == 1 && n.List[0].Binding[0] == '*') {
		s += " " + n.List[0].String()
	} else if 0 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String()
			}
		}
		s += " }"
	}
	if n.Module != nil {
		s += " from " + string(n.Module)
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n ExportStmt) JS() string {
	s := "export"
	if n.Decl != nil {
		if n.Default {
			s += " default"
		}
		return s + " " + n.Decl.JS()
	} else if len(n.List) == 1 && (len(n.List[0].Name) == 1 && n.List[0].Name[0] == '*' || n.List[0].Name == nil && len(n.List[0].Binding) == 1 && n.List[0].Binding[0] == '*') {
		s += " " + n.List[0].JS()
	} else if 0 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.JS()
			}
		}
		s += " }"
	}
	if n.Module != nil {
		s += " from " + string(n.Module)
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ExportStmt) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("export"))
	i += wn
	if err != nil {
		return
	}
	if n.Decl != nil {
		if n.Default {
			wn, err = w.Write([]byte(" default"))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Decl.JSWriteTo(w)
		i += wn
		return
	} else if len(n.List) == 1 && (len(n.List[0].Name) == 1 && n.List[0].Name[0] == '*' || n.List[0].Name == nil && len(n.List[0].Binding) == 1 && n.List[0].Binding[0] == '*') {
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.List[0].JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	} else if 0 < len(n.List) {
		wn, err = w.Write([]byte(" {"))
		i += wn
		if err != nil {
			return
		}
		for j, item := range n.List {
			if j != 0 {
				wn, err = w.Write([]byte(" ,"))
				i += wn
				if err != nil {
					return
				}
			}
			if item.Binding != nil {
				wn, err = w.Write([]byte(" "))
				i += wn
				if err != nil {
					return
				}
				wn, err = item.JSWriteTo(w)
				i += wn
				if err != nil {
					return
				}
			}
		}
		wn, err = w.Write([]byte(" }"))
		i += wn
		if err != nil {
			return
		}
	}
	if n.Module != nil {
		wn, err = w.Write([]byte(" from "))
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write(n.Module)
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// DirectivePrologueStmt is a string literal at the beginning of a function or module (usually "use strict").
type DirectivePrologueStmt struct {
	Value []byte
}

func (n DirectivePrologueStmt) String() string {
	return "Stmt(" + string(n.Value) + ")"
}

// JS converts the node back to valid JavaScript
func (n DirectivePrologueStmt) JS() string {
	return string(n.Value)
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n DirectivePrologueStmt) JSWriteTo(w io.Writer) (i int, err error) {
	return w.Write(n.Value)
}

func (n Comment) stmtNode()               {}
func (n BlockStmt) stmtNode()             {}
func (n EmptyStmt) stmtNode()             {}
func (n ExprStmt) stmtNode()              {}
func (n IfStmt) stmtNode()                {}
func (n DoWhileStmt) stmtNode()           {}
func (n WhileStmt) stmtNode()             {}
func (n ForStmt) stmtNode()               {}
func (n ForInStmt) stmtNode()             {}
func (n ForOfStmt) stmtNode()             {}
func (n SwitchStmt) stmtNode()            {}
func (n BranchStmt) stmtNode()            {}
func (n ReturnStmt) stmtNode()            {}
func (n WithStmt) stmtNode()              {}
func (n LabelledStmt) stmtNode()          {}
func (n ThrowStmt) stmtNode()             {}
func (n TryStmt) stmtNode()               {}
func (n DebuggerStmt) stmtNode()          {}
func (n ImportStmt) stmtNode()            {}
func (n ExportStmt) stmtNode()            {}
func (n DirectivePrologueStmt) stmtNode() {}

////////////////////////////////////////////////////////////////

// PropertyName is a property name for binding properties, method names, and in object literals.
type PropertyName struct {
	Literal  LiteralExpr
	Computed IExpr // can be nil
}

// IsSet returns true is PropertyName is not nil.
func (n PropertyName) IsSet() bool {
	return n.IsComputed() || n.Literal.TokenType != ErrorToken
}

// IsComputed returns true if PropertyName is computed.
func (n PropertyName) IsComputed() bool {
	return n.Computed != nil
}

// IsIdent returns true if PropertyName equals the given identifier name.
func (n PropertyName) IsIdent(data []byte) bool {
	return !n.IsComputed() && n.Literal.TokenType == IdentifierToken && bytes.Equal(data, n.Literal.Data)
}

func (n PropertyName) String() string {
	if n.Computed != nil {
		val := n.Computed.String()
		if val[0] == '(' {
			return "[" + val[1:len(val)-1] + "]"
		}
		return "[" + val + "]"
	}
	return string(n.Literal.Data)
}

// JS converts the node back to valid JavaScript
func (n PropertyName) JS() string {
	if n.Computed != nil {
		return "[" + n.Computed.JS() + "]"
	}
	return string(n.Literal.Data)
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n PropertyName) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Computed != nil {
		wn, err = w.Write([]byte("["))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Computed.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write([]byte("]"))
		i += wn
		return
	}
	wn, err = w.Write(n.Literal.Data)
	i += wn
	return
}

// BindingArray is an array binding pattern.
type BindingArray struct {
	List []BindingElement
	Rest IBinding // can be nil
}

func (n BindingArray) String() string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		s += " " + item.String()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + n.Rest.String() + ")"
	}
	return s + " ]"
}

// JS converts the node back to valid JavaScript
func (n BindingArray) JS() string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.JS()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "..." + n.Rest.JS()
	}
	return s + "]"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n BindingArray) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("["))
	i += wn
	if err != nil {
		return
	}
	for j, item := range n.List {
		if j != 0 {
			wn, err = w.Write([]byte(", "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = item.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			wn, err = w.Write([]byte(", "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = w.Write([]byte("..."))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Rest.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte("]"))
	i += wn
	return
}

// BindingObjectItem is a binding property.
type BindingObjectItem struct {
	Key   *PropertyName // can be nil
	Value BindingElement
}

func (n BindingObjectItem) String() string {
	s := ""
	if n.Key != nil {
		if v, ok := n.Value.Binding.(*Var); !ok || !n.Key.IsIdent(v.Data) {
			s += " " + n.Key.String() + ":"
		}
	}
	return s + " " + n.Value.String()
}

// JS converts the node back to valid JavaScript
func (n BindingObjectItem) JS() string {
	s := ""
	if n.Key != nil {
		if v, ok := n.Value.Binding.(*Var); !ok || !n.Key.IsIdent(v.Data) {
			s += n.Key.JS() + ": "
		}
	}
	return s + n.Value.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n BindingObjectItem) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Key != nil {
		if v, ok := n.Value.Binding.(*Var); !ok || !n.Key.IsIdent(v.Data) {
			wn, err = n.Key.JSWriteTo(w)
			i += wn
			if err != nil {
				return
			}
			wn, err = w.Write([]byte(": "))
			i += wn
			if err != nil {
				return
			}
		}
	}
	wn, err = n.Value.JSWriteTo(w)
	i += wn
	return
}

// BindingObject is an object binding pattern.
type BindingObject struct {
	List []BindingObjectItem
	Rest *Var // can be nil
}

func (n BindingObject) String() string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		s += item.String()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + string(n.Rest.Data) + ")"
	}
	return s + " }"
}

// JS converts the node back to valid JavaScript
func (n BindingObject) JS() string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.JS()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "..." + string(n.Rest.Data)
	}
	return s + "}"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n BindingObject) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("{"))
	i += wn
	if err != nil {
		return
	}
	for j, item := range n.List {
		if j != 0 {
			wn, err = w.Write([]byte(", "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = item.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			wn, err = w.Write([]byte(", "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = w.Write([]byte("..."))
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write(n.Rest.Data)
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte("}"))
	i += wn
	return
}

// BindingElement is a binding element.
type BindingElement struct {
	Binding IBinding // can be nil (in case of ellision)
	Default IExpr    // can be nil
}

func (n BindingElement) String() string {
	if n.Binding == nil {
		return "Binding()"
	}
	s := "Binding(" + n.Binding.String()
	if n.Default != nil {
		s += " = " + n.Default.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n BindingElement) JS() string {
	if n.Binding == nil {
		return ""
	}
	s := n.Binding.JS()
	if n.Default != nil {
		s += " = " + n.Default.JS()
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n BindingElement) JSWriteTo(w io.Writer) (i int, err error) {
	if n.Binding == nil {
		return
	}
	var wn int
	wn, err = n.Binding.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	if n.Default != nil {
		wn, err = w.Write([]byte(" = "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Default.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	return
}

func (v *Var) bindingNode()          {}
func (n BindingArray) bindingNode()  {}
func (n BindingObject) bindingNode() {}

////////////////////////////////////////////////////////////////

// VarDecl is a variable statement or lexical declaration.
type VarDecl struct {
	TokenType
	List             []BindingElement
	Scope            *Scope
	InFor, InForInOf bool
}

func (n VarDecl) String() string {
	s := "Decl(" + n.TokenType.String()
	for _, item := range n.List {
		s += " " + item.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n VarDecl) JS() string {
	s := n.TokenType.String()
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		s += " " + item.JS()
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n VarDecl) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write(n.TokenType.Bytes())
	i += wn
	if err != nil {
		return
	}
	for j, item := range n.List {
		if j != 0 {
			wn, err = w.Write([]byte(","))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = item.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// Params is a list of parameters for functions, methods, and arrow function.
type Params struct {
	List []BindingElement
	Rest IBinding // can be nil
}

func (n Params) String() string {
	s := "Params("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "...Binding(" + n.Rest.String() + ")"
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n Params) JS() string {
	s := "("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.JS()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "..." + n.Rest.JS()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n Params) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("("))
	i += wn
	if err != nil {
		return
	}
	for j, item := range n.List {
		if j != 0 {
			wn, err = w.Write([]byte(", "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = item.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			wn, err = w.Write([]byte(", "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = w.Write([]byte("..."))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Rest.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte(")"))
	i += wn
	return
}

// FuncDecl is an (async) (generator) function declaration or expression.
type FuncDecl struct {
	Async     bool
	Generator bool
	Name      *Var // can be nil
	Params    Params
	Body      BlockStmt
}

func (n FuncDecl) String() string {
	s := "Decl("
	if n.Async {
		s += "async function"
	} else {
		s += "function"
	}
	if n.Generator {
		s += "*"
	}
	if n.Name != nil {
		s += " " + string(n.Name.Data)
	}
	return s + " " + n.Params.String() + " " + n.Body.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n FuncDecl) JS() string {
	s := ""
	if n.Async {
		s += "async function"
	} else {
		s += "function"
	}
	if n.Generator {
		s += "*"
	}
	if n.Name != nil {
		s += " " + string(n.Name.Data)
	}
	return s + " " + n.Params.JS() + " " + n.Body.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n FuncDecl) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Async {
		wn, err = w.Write([]byte("async function"))
	} else {
		wn, err = w.Write([]byte("function"))
	}
	i += wn
	if err != nil {
		return
	}

	if n.Generator {
		wn, err = w.Write([]byte("*"))
		i += wn
		if err != nil {
			return
		}
	}
	if n.Name != nil {
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write(n.Name.Data)
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte(" "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Params.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(" "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Body.JSWriteTo(w)
	i += wn
	return
}

// MethodDecl is a method definition in a class declaration.
type MethodDecl struct {
	Static    bool
	Async     bool
	Generator bool
	Get       bool
	Set       bool
	Name      PropertyName
	Params    Params
	Body      BlockStmt
}

func (n MethodDecl) String() string {
	s := ""
	if n.Static {
		s += " static"
	}
	if n.Async {
		s += " async"
	}
	if n.Generator {
		s += " *"
	}
	if n.Get {
		s += " get"
	}
	if n.Set {
		s += " set"
	}
	s += " " + n.Name.String() + " " + n.Params.String() + " " + n.Body.String()
	return "Method(" + s[1:] + ")"
}

// JS converts the node back to valid JavaScript
func (n MethodDecl) JS() string {
	s := ""
	if n.Static {
		s += " static"
	}
	if n.Async {
		s += " async"
	}
	if n.Generator {
		s += " *"
	}
	if n.Get {
		s += " get"
	}
	if n.Set {
		s += " set"
	}
	s += " " + n.Name.JS() + " " + n.Params.JS() + " " + n.Body.JS()
	return s[1:]
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n MethodDecl) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Static {
		wn, err = w.Write([]byte("static"))
		i += wn
		if err != nil {
			return
		}
	}
	if n.Async {
		if wn > 0 {
			wn, err = w.Write([]byte(" "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = w.Write([]byte("async"))
		i += wn
		if err != nil {
			return
		}
	}
	if n.Generator {
		if wn > 0 {
			wn, err = w.Write([]byte(" "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = w.Write([]byte("*"))
		i += wn
		if err != nil {
			return
		}
	}
	if n.Get {
		if wn > 0 {
			wn, err = w.Write([]byte(" "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = w.Write([]byte("get"))
		i += wn
		if err != nil {
			return
		}
	}
	if n.Set {
		if wn > 0 {
			wn, err = w.Write([]byte(" "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = w.Write([]byte("set"))
		i += wn
		if err != nil {
			return
		}
	}
	if wn > 0 {
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = n.Name.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(" "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Params.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(" "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Body.JSWriteTo(w)
	i += wn
	return
}

// Field is a field definition in a class declaration.
type Field struct {
	Static bool
	Name   PropertyName
	Init   IExpr
}

func (n Field) String() string {
	s := "Field("
	if n.Static {
		s += "static "
	}
	s += n.Name.String()
	if n.Init != nil {
		s += " = " + n.Init.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n Field) JS() string {
	s := ""
	if n.Static {
		s += "static "
	}
	s += n.Name.String()
	if n.Init != nil {
		s += " = " + n.Init.JS()
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n Field) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Static {
		wn, err = w.Write([]byte("static "))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = n.Name.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	if n.Init != nil {
		wn, err = w.Write([]byte(" = "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Init.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// ClassElement is a class element that is either a static block, a field definition, or a class method
type ClassElement struct {
	StaticBlock *BlockStmt  // can be nil
	Method      *MethodDecl // can be nil
	Field
}

func (n ClassElement) String() string {
	if n.StaticBlock != nil {
		return "Static(" + n.StaticBlock.String() + ")"
	} else if n.Method != nil {
		return n.Method.String()
	}
	return n.Field.String()
}

// JS converts the node back to valid JavaScript
func (n ClassElement) JS() string {
	if n.StaticBlock != nil {
		return "static " + n.StaticBlock.JS()
	} else if n.Method != nil {
		return n.Method.JS()
	}
	return n.Field.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ClassElement) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.StaticBlock != nil {
		wn, err = w.Write([]byte("static "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.StaticBlock.JSWriteTo(w)
		i += wn
		return
	} else if n.Method != nil {
		wn, err = n.Method.JSWriteTo(w)
		i += wn
		return
	}
	wn, err = n.Field.JSWriteTo(w)
	i += wn
	return
}

// ClassDecl is a class declaration.
type ClassDecl struct {
	Name    *Var  // can be nil
	Extends IExpr // can be nil
	List    []ClassElement
}

func (n ClassDecl) String() string {
	s := "Decl(class"
	if n.Name != nil {
		s += " " + string(n.Name.Data)
	}
	if n.Extends != nil {
		s += " extends " + n.Extends.String()
	}
	for _, item := range n.List {
		s += " " + item.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n ClassDecl) JS() string {
	s := "class"
	if n.Name != nil {
		s += " " + string(n.Name.Data)
	}
	if n.Extends != nil {
		s += " extends " + n.Extends.JS()
	}
	s += " { "
	for _, item := range n.List {
		s += item.JS() + "; "
	}
	return s + "}"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ClassDecl) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("class"))
	i += wn
	if err != nil {
		return
	}
	if n.Name != nil {
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write(n.Name.Data)
		i += wn
		if err != nil {
			return
		}
	}
	if n.Extends != nil {
		wn, err = w.Write([]byte(" extends "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Extends.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte(" { "))
	i += wn
	if err != nil {
		return
	}
	for _, item := range n.List {
		wn, err = item.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write([]byte("; "))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte("}"))
	i += wn
	return
}

func (n VarDecl) stmtNode()   {}
func (n FuncDecl) stmtNode()  {}
func (n ClassDecl) stmtNode() {}

func (n VarDecl) exprNode()    {} // not a real IExpr, used for ForInit and ExportDecl
func (n FuncDecl) exprNode()   {}
func (n ClassDecl) exprNode()  {}
func (n MethodDecl) exprNode() {} // not a real IExpr, used for ObjectExpression PropertyName

////////////////////////////////////////////////////////////////

// LiteralExpr can be this, null, boolean, numeric, string, or regular expression literals.
type LiteralExpr struct {
	TokenType
	Data []byte
}

func (n LiteralExpr) String() string {
	return string(n.Data)
}

// JS converts the node back to valid JavaScript
func (n LiteralExpr) JS() string {
	return string(n.Data)
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n LiteralExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write(n.Data)
	i += wn
	return
}

// JSON converts the node back to valid JSON
func (n LiteralExpr) JSON(buf *bytes.Buffer) error {
	if n.TokenType == TrueToken || n.TokenType == FalseToken || n.TokenType == NullToken || n.TokenType == DecimalToken {
		buf.Write(n.Data)
		return nil
	} else if n.TokenType == StringToken {
		data := n.Data
		if n.Data[0] == '\'' {
			data = parse.Copy(data)
			data = bytes.ReplaceAll(data, []byte(`"`), []byte(`\"`))
			data[0] = '"'
			data[len(data)-1] = '"'
		}
		buf.Write(data)
		return nil
	}
	return ErrInvalidJSON
}

// Element is an array literal element.
type Element struct {
	Value  IExpr // can be nil
	Spread bool
}

func (n Element) String() string {
	s := ""
	if n.Value != nil {
		if n.Spread {
			s += "..."
		}
		s += n.Value.String()
	}
	return s
}

// JS converts the node back to valid JavaScript
func (n Element) JS() string {
	s := ""
	if n.Value != nil {
		if n.Spread {
			s += "..."
		}
		s += n.Value.JS()
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n Element) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Value != nil {
		if n.Spread {
			wn, err = w.Write([]byte("..."))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = n.Value.JSWriteTo(w)
		i += wn
	}
	return
}

// ArrayExpr is an array literal.
type ArrayExpr struct {
	List []Element
}

func (n ArrayExpr) String() string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		if item.Value != nil {
			if item.Spread {
				s += "..."
			}
			s += item.Value.String()
		}
	}
	if 0 < len(n.List) && n.List[len(n.List)-1].Value == nil {
		s += ","
	}
	return s + "]"
}

// JS converts the node back to valid JavaScript
func (n ArrayExpr) JS() string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		if item.Value != nil {
			if item.Spread {
				s += "..."
			}
			s += item.Value.JS()
		}
	}
	if 0 < len(n.List) && n.List[len(n.List)-1].Value == nil {
		s += ","
	}
	return s + "]"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ArrayExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("["))
	i += wn
	if err != nil {
		return
	}
	for j, item := range n.List {
		if j != 0 {
			wn, err = w.Write([]byte(", "))
			i += wn
			if err != nil {
				return
			}
		}
		if item.Value != nil {
			if item.Spread {
				wn, err = w.Write([]byte("..."))
				i += wn
				if err != nil {
					return
				}
			}
			wn, err = item.Value.JSWriteTo(w)
			i += wn
			if err != nil {
				return
			}
		}
	}
	if 0 < len(n.List) && n.List[len(n.List)-1].Value == nil {
		wn, err = w.Write([]byte(","))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte("]"))
	i += wn
	return
}

// JSON converts the node back to valid JSON
func (n ArrayExpr) JSON(buf *bytes.Buffer) error {
	buf.WriteByte('[')
	for i, item := range n.List {
		if i != 0 {
			buf.WriteString(", ")
		}
		if item.Value == nil || item.Spread {
			return ErrInvalidJSON
		}
		val, ok := item.Value.(JSONer)
		if !ok {
			return ErrInvalidJSON
		} else if err := val.JSON(buf); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

// Property is a property definition in an object literal.
type Property struct {
	// either Name or Spread are set. When Spread is set then Value is AssignmentExpression
	// if Init is set then Value is IdentifierReference, otherwise it can also be MethodDefinition
	Name   *PropertyName // can be nil
	Spread bool
	Value  IExpr
	Init   IExpr // can be nil
}

func (n Property) String() string {
	s := ""
	if n.Name != nil {
		if v, ok := n.Value.(*Var); !ok || !n.Name.IsIdent(v.Data) {
			s += n.Name.String() + ": "
		}
	} else if n.Spread {
		s += "..."
	}
	s += n.Value.String()
	if n.Init != nil {
		s += " = " + n.Init.String()
	}
	return s
}

// JS converts the node back to valid JavaScript
func (n Property) JS() string {
	s := ""
	if n.Name != nil {
		if v, ok := n.Value.(*Var); !ok || !n.Name.IsIdent(v.Data) {
			s += n.Name.JS() + ": "
		}
	} else if n.Spread {
		s += "..."
	}
	s += n.Value.JS()
	if n.Init != nil {
		s += " = " + n.Init.JS()
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n Property) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Name != nil {
		if v, ok := n.Value.(*Var); !ok || !n.Name.IsIdent(v.Data) {
			wn, err = n.Name.JSWriteTo(w)
			i += wn
			if err != nil {
				return
			}
			wn, err = w.Write([]byte(": "))
			i += wn
			if err != nil {
				return
			}
		}
	} else if n.Spread {
		wn, err = w.Write([]byte("..."))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = n.Value.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	if n.Init != nil {
		wn, err = w.Write([]byte(" = "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Init.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// JSON converts the node back to valid JSON
func (n Property) JSON(buf *bytes.Buffer) error {
	if n.Name == nil || n.Name.Literal.TokenType != StringToken && n.Name.Literal.TokenType != IdentifierToken || n.Spread || n.Init != nil {
		return ErrInvalidJSON
	} else if n.Name.Literal.TokenType == IdentifierToken {
		buf.WriteByte('"')
		buf.Write(n.Name.Literal.Data)
		buf.WriteByte('"')
	} else {
		_ = n.Name.Literal.JSON(buf)
	}
	buf.WriteString(": ")

	val, ok := n.Value.(JSONer)
	if !ok {
		return ErrInvalidJSON
	} else if err := val.JSON(buf); err != nil {
		return err
	}
	return nil
}

// ObjectExpr is an object literal.
type ObjectExpr struct {
	List []Property
}

func (n ObjectExpr) String() string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String()
	}
	return s + "}"
}

// JS converts the node back to valid JavaScript
func (n ObjectExpr) JS() string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.JS()
	}
	return s + "}"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ObjectExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("{"))
	i += wn
	if err != nil {
		return
	}
	for j, item := range n.List {
		if j != 0 {
			wn, err = w.Write([]byte(", "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = item.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte("}"))
	i += wn
	return
}

// JSON converts the node back to valid JSON
func (n ObjectExpr) JSON(buf *bytes.Buffer) error {
	buf.WriteByte('{')
	for i, item := range n.List {
		if i != 0 {
			buf.WriteString(", ")
		}
		if err := item.JSON(buf); err != nil {
			return err
		}
	}
	buf.WriteByte('}')
	return nil
}

// TemplatePart is a template head or middle.
type TemplatePart struct {
	Value []byte
	Expr  IExpr
}

func (n TemplatePart) String() string {
	return string(n.Value) + n.Expr.String()
}

// JS converts the node back to valid JavaScript
func (n TemplatePart) JS() string {
	return string(n.Value) + n.Expr.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n TemplatePart) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write(n.Value)
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Expr.JSWriteTo(w)
	i += wn
	return
}

// TemplateExpr is a template literal or member/call expression, super property, or optional chain with template literal.
type TemplateExpr struct {
	Tag      IExpr // can be nil
	List     []TemplatePart
	Tail     []byte
	Prec     OpPrec
	Optional bool
}

func (n TemplateExpr) String() string {
	s := ""
	if n.Tag != nil {
		s += n.Tag.String()
		if n.Optional {
			s += "?."
		}
	}
	for _, item := range n.List {
		s += item.String()
	}
	return s + string(n.Tail)
}

// JS converts the node back to valid JavaScript
func (n TemplateExpr) JS() string {
	s := ""
	if n.Tag != nil {
		s += n.Tag.JS()
		if n.Optional {
			s += "?."
		}
	}
	for _, item := range n.List {
		s += item.JS()
	}
	return s + string(n.Tail)
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n TemplateExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Tag != nil {
		wn, err = n.Tag.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
		if n.Optional {
			wn, err = w.Write([]byte("?."))
			i += wn
			if err != nil {
				return
			}
		}
	}
	for _, item := range n.List {
		wn, err = item.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write(n.Tail)
	i += wn
	return
}

// GroupExpr is a parenthesized expression.
type GroupExpr struct {
	X IExpr
}

func (n GroupExpr) String() string {
	return "(" + n.X.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n GroupExpr) JS() string {
	return "(" + n.X.JS() + ")"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n GroupExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("("))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.X.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(")"))
	i += wn
	return
}

// IndexExpr is a member/call expression, super property, or optional chain with an index expression.
type IndexExpr struct {
	X        IExpr
	Y        IExpr
	Prec     OpPrec
	Optional bool
}

func (n IndexExpr) String() string {
	if n.Optional {
		return "(" + n.X.String() + "?.[" + n.Y.String() + "])"
	}
	return "(" + n.X.String() + "[" + n.Y.String() + "])"
}

// JS converts the node back to valid JavaScript
func (n IndexExpr) JS() string {
	if n.Optional {
		return n.X.JS() + "?.[" + n.Y.JS() + "]"
	}
	return n.X.JS() + "[" + n.Y.JS() + "]"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n IndexExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = n.X.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	if n.Optional {
		wn, err = w.Write([]byte("?.["))
		i += wn
		if err != nil {
			return
		}
	} else {
		wn, err = w.Write([]byte("["))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = n.Y.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte("]"))
	i += wn
	return
}

// DotExpr is a member/call expression, super property, or optional chain with a dot expression.
type DotExpr struct {
	X        IExpr
	Y        LiteralExpr
	Prec     OpPrec
	Optional bool
}

func (n DotExpr) String() string {
	if n.Optional {
		return "(" + n.X.String() + "?." + n.Y.String() + ")"
	}
	return "(" + n.X.String() + "." + n.Y.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n DotExpr) JS() string {
	if n.Optional {
		return n.X.JS() + "?." + n.Y.JS()
	}
	return n.X.JS() + "." + n.Y.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n DotExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = n.X.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	if n.Optional {
		wn, err = w.Write([]byte("?."))
		i += wn
		if err != nil {
			return
		}
	} else {
		wn, err = w.Write([]byte("."))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = n.Y.JSWriteTo(w)
	i += wn
	return
}

// NewTargetExpr is a new target meta property.
type NewTargetExpr struct {
}

func (n NewTargetExpr) String() string {
	return "(new.target)"
}

// JS converts the node back to valid JavaScript
func (n NewTargetExpr) JS() string {
	return "new.target"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n NewTargetExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("new.target"))
	i += wn
	return
}

// ImportMetaExpr is a import meta meta property.
type ImportMetaExpr struct {
}

func (n ImportMetaExpr) String() string {
	return "(import.meta)"
}

// JS converts the node back to valid JavaScript
func (n ImportMetaExpr) JS() string {
	return "import.meta"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ImportMetaExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("import.meta"))
	i += wn
	return
}

type Arg struct {
	Value IExpr
	Rest  bool
}

func (n Arg) String() string {
	s := ""
	if n.Rest {
		s += "..."
	}
	return s + n.Value.String()
}

// JS converts the node back to valid JavaScript
func (n Arg) JS() string {
	s := ""
	if n.Rest {
		s += "..."
	}
	return s + n.Value.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n Arg) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Rest {
		wn, err = w.Write([]byte("..."))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = n.Value.JSWriteTo(w)
	i += wn
	return
}

// Args is a list of arguments as used by new and call expressions.
type Args struct {
	List []Arg
}

func (n Args) String() string {
	s := "("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n Args) JS() string {
	s := ""
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.JS()
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n Args) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	for j, item := range n.List {
		if j != 0 {
			wn, err = w.Write([]byte(", "))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = item.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// NewExpr is a new expression or new member expression.
type NewExpr struct {
	X    IExpr
	Args *Args // can be nil
}

func (n NewExpr) String() string {
	if n.Args != nil {
		return "(new " + n.X.String() + n.Args.String() + ")"
	}
	return "(new " + n.X.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n NewExpr) JS() string {
	if n.Args != nil {
		return "new " + n.X.JS() + "(" + n.Args.JS() + ")"
	}

	// always use parentheses to prevent errors when chaining e.g. new Date().getTime()
	return "new " + n.X.JS() + "()"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n NewExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("new "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.X.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	if n.Args != nil {
		wn, err = w.Write([]byte("("))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.Args.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write([]byte(")"))
		i += wn
		if err != nil {
			return
		}
	} else {
		wn, err = w.Write([]byte("()"))
		i += wn
		if err != nil {
			return
		}
	}
	return
}

// CallExpr is a call expression.
type CallExpr struct {
	X        IExpr
	Args     Args
	Optional bool
}

func (n CallExpr) String() string {
	if n.Optional {
		return "(" + n.X.String() + "?." + n.Args.String() + ")"
	}
	return "(" + n.X.String() + n.Args.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n CallExpr) JS() string {
	if n.Optional {
		return n.X.JS() + "?.(" + n.Args.JS() + ")"
	}
	return n.X.JS() + "(" + n.Args.JS() + ")"
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n CallExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = n.X.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	if n.Optional {
		wn, err = w.Write([]byte("?.("))
		i += wn
		if err != nil {
			return
		}
	} else {
		wn, err = w.Write([]byte("("))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = n.Args.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(")"))
	i += wn
	if err != nil {
		return
	}
	return
}

// UnaryExpr is an update or unary expression.
type UnaryExpr struct {
	Op TokenType
	X  IExpr
}

func (n UnaryExpr) String() string {
	if n.Op == PostIncrToken || n.Op == PostDecrToken {
		return "(" + n.X.String() + n.Op.String() + ")"
	} else if IsIdentifierName(n.Op) {
		return "(" + n.Op.String() + " " + n.X.String() + ")"
	}
	return "(" + n.Op.String() + n.X.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n UnaryExpr) JS() string {
	if n.Op == PostIncrToken || n.Op == PostDecrToken {
		return n.X.JS() + n.Op.String()
	} else if unary, ok := n.X.(*UnaryExpr); ok && (n.Op == PosToken && unary.Op == PreIncrToken || n.Op == NegToken && unary.Op == PreDecrToken) || IsIdentifierName(n.Op) {
		return n.Op.String() + " " + n.X.JS()
	}
	return n.Op.String() + n.X.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n UnaryExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Op == PostIncrToken || n.Op == PostDecrToken {
		wn, err = n.X.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write(n.Op.Bytes())
		i += wn
		return
	} else if unary, ok := n.X.(UnaryExpr); ok && (n.Op == PosToken && unary.Op == PreIncrToken || n.Op == NegToken && unary.Op == PreDecrToken) || IsIdentifierName(n.Op) {
		wn, err = w.Write(n.Op.Bytes())
		i += wn
		if err != nil {
			return
		}
		wn, err = w.Write([]byte(" "))
		i += wn
		if err != nil {
			return
		}
		wn, err = n.X.JSWriteTo(w)
		i += wn
		return
	}
	wn, err = w.Write(n.Op.Bytes())
	i += wn
	if err != nil {
		return
	}
	wn, err = n.X.JSWriteTo(w)
	i += wn
	return
}

// JSON converts the node back to valid JSON
func (n UnaryExpr) JSON(buf *bytes.Buffer) error {
	if lit, ok := n.X.(*LiteralExpr); ok && n.Op == NegToken && lit.TokenType == DecimalToken {
		buf.WriteByte('-')
		buf.Write(lit.Data)
		return nil
	}
	return ErrInvalidJSON
}

// BinaryExpr is a binary expression.
type BinaryExpr struct {
	Op   TokenType
	X, Y IExpr
}

func (n BinaryExpr) String() string {
	if IsIdentifierName(n.Op) {
		return "(" + n.X.String() + " " + n.Op.String() + " " + n.Y.String() + ")"
	}
	return "(" + n.X.String() + n.Op.String() + n.Y.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n BinaryExpr) JS() string {
	return n.X.JS() + " " + n.Op.String() + " " + n.Y.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n BinaryExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = n.X.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(" "))
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write(n.Op.Bytes())
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(" "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Y.JSWriteTo(w)
	i += wn
	return
}

// CondExpr is a conditional expression.
type CondExpr struct {
	Cond, X, Y IExpr
}

func (n CondExpr) String() string {
	return "(" + n.Cond.String() + " ? " + n.X.String() + " : " + n.Y.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n CondExpr) JS() string {
	return n.Cond.JS() + " ? " + n.X.JS() + " : " + n.Y.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n CondExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = n.Cond.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(" ? "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.X.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(" : "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Y.JSWriteTo(w)
	i += wn
	return
}

// YieldExpr is a yield expression.
type YieldExpr struct {
	Generator bool
	X         IExpr // can be nil
}

func (n YieldExpr) String() string {
	if n.X == nil {
		return "(yield)"
	}
	s := "(yield"
	if n.Generator {
		s += "*"
	}
	return s + " " + n.X.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n YieldExpr) JS() string {
	if n.X == nil {
		return "yield"
	}
	s := "yield"
	if n.Generator {
		s += "*"
	}
	return s + " " + n.X.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n YieldExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	wn, err = w.Write([]byte("yield"))
	i += wn
	if err != nil {
		return
	}
	if n.X == nil {
		return
	}
	if n.Generator {
		wn, err = w.Write([]byte("*"))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = w.Write([]byte(" "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.X.JSWriteTo(w)
	i += wn
	return
}

// ArrowFunc is an (async) arrow function.
type ArrowFunc struct {
	Async  bool
	Params Params
	Body   BlockStmt
}

func (n ArrowFunc) String() string {
	s := "("
	if n.Async {
		s += "async "
	}
	return s + n.Params.String() + " => " + n.Body.String() + ")"
}

// JS converts the node back to valid JavaScript
func (n ArrowFunc) JS() string {
	s := ""
	if n.Async {
		s += "async "
	}
	return s + n.Params.JS() + " => " + n.Body.JS()
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n ArrowFunc) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	if n.Async {
		wn, err = w.Write([]byte("async "))
		i += wn
		if err != nil {
			return
		}
	}
	wn, err = n.Params.JSWriteTo(w)
	i += wn
	if err != nil {
		return
	}
	wn, err = w.Write([]byte(" => "))
	i += wn
	if err != nil {
		return
	}
	wn, err = n.Body.JSWriteTo(w)
	i += wn
	return
}

// CommaExpr is a series of comma expressions.
type CommaExpr struct {
	List []IExpr
}

func (n CommaExpr) String() string {
	s := "("
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		s += item.String()
	}
	return s + ")"
}

// JS converts the node back to valid JavaScript
func (n CommaExpr) JS() string {
	s := ""
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		s += item.JS()
	}
	return s
}

// JS converts the node back to valid JavaScript (writes to io.Writer)
func (n CommaExpr) JSWriteTo(w io.Writer) (i int, err error) {
	var wn int
	for j, item := range n.List {
		if j != 0 {
			wn, err = w.Write([]byte(","))
			i += wn
			if err != nil {
				return
			}
		}
		wn, err = item.JSWriteTo(w)
		i += wn
		if err != nil {
			return
		}
	}
	return
}

func (v *Var) exprNode()           {}
func (n LiteralExpr) exprNode()    {}
func (n ArrayExpr) exprNode()      {}
func (n ObjectExpr) exprNode()     {}
func (n TemplateExpr) exprNode()   {}
func (n GroupExpr) exprNode()      {}
func (n DotExpr) exprNode()        {}
func (n IndexExpr) exprNode()      {}
func (n NewTargetExpr) exprNode()  {}
func (n ImportMetaExpr) exprNode() {}
func (n NewExpr) exprNode()        {}
func (n CallExpr) exprNode()       {}
func (n UnaryExpr) exprNode()      {}
func (n BinaryExpr) exprNode()     {}
func (n CondExpr) exprNode()       {}
func (n YieldExpr) exprNode()      {}
func (n ArrowFunc) exprNode()      {}
func (n CommaExpr) exprNode()      {}
