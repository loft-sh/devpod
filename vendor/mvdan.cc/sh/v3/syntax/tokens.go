// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

//go:generate stringer -type token -linecomment -trimprefix _

type token uint32

// The list of all possible tokens.
const (
	illegalTok token = iota

	_EOF
	_Newl
	_Lit
	_LitWord
	_LitRedir

	sglQuote // '
	dblQuote // "
	bckQuote // `

	and    // &
	andAnd // &&
	orOr   // ||
	or     // |
	orAnd  // |&

	dollar       // $
	dollSglQuote // $'
	dollDblQuote // $"
	dollBrace    // ${
	dollBrack    // $[
	dollParen    // $(
	dollDblParen // $((
	leftBrack    // [
	dblLeftBrack // [[
	leftParen    // (
	dblLeftParen // ((

	rightBrace    // }
	rightBrack    // ]
	rightParen    // )
	dblRightParen // ))
	semicolon     // ;

	dblSemicolon // ;;
	semiAnd      // ;&
	dblSemiAnd   // ;;&
	semiOr       // ;|

	exclMark // !
	tilde    // ~
	addAdd   // ++
	subSub   // --
	star     // *
	power    // **
	equal    // ==
	nequal   // !=
	lequal   // <=
	gequal   // >=

	addAssgn // +=
	subAssgn // -=
	mulAssgn // *=
	quoAssgn // /=
	remAssgn // %=
	andAssgn // &=
	orAssgn  // |=
	xorAssgn // ^=
	shlAssgn // <<=
	shrAssgn // >>=

	rdrOut   // >
	appOut   // >>
	rdrIn    // <
	rdrInOut // <>
	dplIn    // <&
	dplOut   // >&
	clbOut   // >|
	hdoc     // <<
	dashHdoc // <<-
	wordHdoc // <<<
	rdrAll   // &>
	appAll   // &>>

	cmdIn  // <(
	cmdOut // >(

	plus     // +
	colPlus  // :+
	minus    // -
	colMinus // :-
	quest    // ?
	colQuest // :?
	assgn    // =
	colAssgn // :=
	perc     // %
	dblPerc  // %%
	hash     // #
	dblHash  // ##
	caret    // ^
	dblCaret // ^^
	comma    // ,
	dblComma // ,,
	at       // @
	slash    // /
	dblSlash // //
	colon    // :

	tsExists  // -e
	tsRegFile // -f
	tsDirect  // -d
	tsCharSp  // -c
	tsBlckSp  // -b
	tsNmPipe  // -p
	tsSocket  // -S
	tsSmbLink // -L
	tsSticky  // -k
	tsGIDSet  // -g
	tsUIDSet  // -u
	tsGrpOwn  // -G
	tsUsrOwn  // -O
	tsModif   // -N
	tsRead    // -r
	tsWrite   // -w
	tsExec    // -x
	tsNoEmpty // -s
	tsFdTerm  // -t
	tsEmpStr  // -z
	tsNempStr // -n
	tsOptSet  // -o
	tsVarSet  // -v
	tsRefVar  // -R

	tsReMatch // =~
	tsNewer   // -nt
	tsOlder   // -ot
	tsDevIno  // -ef
	tsEql     // -eq
	tsNeq     // -ne
	tsLeq     // -le
	tsGeq     // -ge
	tsLss     // -lt
	tsGtr     // -gt

	globQuest // ?(
	globStar  // *(
	globPlus  // +(
	globAt    // @(
	globExcl  // !(
)

type RedirOperator token

const (
	RdrOut   = RedirOperator(rdrOut) + iota // >
	AppOut                                  // >>
	RdrIn                                   // <
	RdrInOut                                // <>
	DplIn                                   // <&
	DplOut                                  // >&
	ClbOut                                  // >|
	Hdoc                                    // <<
	DashHdoc                                // <<-
	WordHdoc                                // <<<
	RdrAll                                  // &>
	AppAll                                  // &>>
)

type ProcOperator token

const (
	CmdIn  = ProcOperator(cmdIn) + iota // <(
	CmdOut                              // >(
)

type GlobOperator token

const (
	GlobZeroOrOne  = GlobOperator(globQuest) + iota // ?(
	GlobZeroOrMore                                  // *(
	GlobOneOrMore                                   // +(
	GlobOne                                         // @(
	GlobExcept                                      // !(
)

type BinCmdOperator token

const (
	AndStmt = BinCmdOperator(andAnd) + iota // &&
	OrStmt                                  // ||
	Pipe                                    // |
	PipeAll                                 // |&
)

type CaseOperator token

const (
	Break       = CaseOperator(dblSemicolon) + iota // ;;
	Fallthrough                                     // ;&
	Resume                                          // ;;&
	ResumeKorn                                      // ;|
)

type ParNamesOperator token

const (
	NamesPrefix      = ParNamesOperator(star) // *
	NamesPrefixWords = ParNamesOperator(at)   // @
)

type ParExpOperator token

const (
	AlternateUnset       = ParExpOperator(plus) + iota // +
	AlternateUnsetOrNull                               // :+
	DefaultUnset                                       // -
	DefaultUnsetOrNull                                 // :-
	ErrorUnset                                         // ?
	ErrorUnsetOrNull                                   // :?
	AssignUnset                                        // =
	AssignUnsetOrNull                                  // :=
	RemSmallSuffix                                     // %
	RemLargeSuffix                                     // %%
	RemSmallPrefix                                     // #
	RemLargePrefix                                     // ##
	UpperFirst                                         // ^
	UpperAll                                           // ^^
	LowerFirst                                         // ,
	LowerAll                                           // ,,
	OtherParamOps                                      // @
)

type UnAritOperator token

const (
	Not         = UnAritOperator(exclMark) + iota // !
	BitNegation                                   // ~
	Inc                                           // ++
	Dec                                           // --
	Plus        = UnAritOperator(plus)            // +
	Minus       = UnAritOperator(minus)           // -
)

type BinAritOperator token

const (
	Add = BinAritOperator(plus)   // +
	Sub = BinAritOperator(minus)  // -
	Mul = BinAritOperator(star)   // *
	Quo = BinAritOperator(slash)  // /
	Rem = BinAritOperator(perc)   // %
	Pow = BinAritOperator(power)  // **
	Eql = BinAritOperator(equal)  // ==
	Gtr = BinAritOperator(rdrOut) // >
	Lss = BinAritOperator(rdrIn)  // <
	Neq = BinAritOperator(nequal) // !=
	Leq = BinAritOperator(lequal) // <=
	Geq = BinAritOperator(gequal) // >=
	And = BinAritOperator(and)    // &
	Or  = BinAritOperator(or)     // |
	Xor = BinAritOperator(caret)  // ^
	Shr = BinAritOperator(appOut) // >>
	Shl = BinAritOperator(hdoc)   // <<

	AndArit   = BinAritOperator(andAnd) // &&
	OrArit    = BinAritOperator(orOr)   // ||
	Comma     = BinAritOperator(comma)  // ,
	TernQuest = BinAritOperator(quest)  // ?
	TernColon = BinAritOperator(colon)  // :

	Assgn    = BinAritOperator(assgn)    // =
	AddAssgn = BinAritOperator(addAssgn) // +=
	SubAssgn = BinAritOperator(subAssgn) // -=
	MulAssgn = BinAritOperator(mulAssgn) // *=
	QuoAssgn = BinAritOperator(quoAssgn) // /=
	RemAssgn = BinAritOperator(remAssgn) // %=
	AndAssgn = BinAritOperator(andAssgn) // &=
	OrAssgn  = BinAritOperator(orAssgn)  // |=
	XorAssgn = BinAritOperator(xorAssgn) // ^=
	ShlAssgn = BinAritOperator(shlAssgn) // <<=
	ShrAssgn = BinAritOperator(shrAssgn) // >>=
)

type UnTestOperator token

const (
	TsExists  = UnTestOperator(tsExists) + iota // -e
	TsRegFile                                   // -f
	TsDirect                                    // -d
	TsCharSp                                    // -c
	TsBlckSp                                    // -b
	TsNmPipe                                    // -p
	TsSocket                                    // -S
	TsSmbLink                                   // -L
	TsSticky                                    // -k
	TsGIDSet                                    // -g
	TsUIDSet                                    // -u
	TsGrpOwn                                    // -G
	TsUsrOwn                                    // -O
	TsModif                                     // -N
	TsRead                                      // -r
	TsWrite                                     // -w
	TsExec                                      // -x
	TsNoEmpty                                   // -s
	TsFdTerm                                    // -t
	TsEmpStr                                    // -z
	TsNempStr                                   // -n
	TsOptSet                                    // -o
	TsVarSet                                    // -v
	TsRefVar                                    // -R
	TsNot     = UnTestOperator(exclMark)        // !
)

type BinTestOperator token

const (
	TsReMatch    = BinTestOperator(tsReMatch) + iota // =~
	TsNewer                                          // -nt
	TsOlder                                          // -ot
	TsDevIno                                         // -ef
	TsEql                                            // -eq
	TsNeq                                            // -ne
	TsLeq                                            // -le
	TsGeq                                            // -ge
	TsLss                                            // -lt
	TsGtr                                            // -gt
	AndTest      = BinTestOperator(andAnd)           // &&
	OrTest       = BinTestOperator(orOr)             // ||
	TsMatchShort = BinTestOperator(assgn)            // =
	TsMatch      = BinTestOperator(equal)            // ==
	TsNoMatch    = BinTestOperator(nequal)           // !=
	TsBefore     = BinTestOperator(rdrIn)            // <
	TsAfter      = BinTestOperator(rdrOut)           // >
)

func (o RedirOperator) String() string    { return token(o).String() }
func (o ProcOperator) String() string     { return token(o).String() }
func (o GlobOperator) String() string     { return token(o).String() }
func (o BinCmdOperator) String() string   { return token(o).String() }
func (o CaseOperator) String() string     { return token(o).String() }
func (o ParNamesOperator) String() string { return token(o).String() }
func (o ParExpOperator) String() string   { return token(o).String() }
func (o UnAritOperator) String() string   { return token(o).String() }
func (o BinAritOperator) String() string  { return token(o).String() }
func (o UnTestOperator) String() string   { return token(o).String() }
func (o BinTestOperator) String() string  { return token(o).String() }
