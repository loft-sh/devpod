package syntax

// compact specifies whether we allow spaces between expressions.
// This is true for let
func (p *Parser) arithmExpr(compact bool) ArithmExpr {
	return p.arithmExprComma(compact)
}

// These function names are inspired by Bash's expr.c

func (p *Parser) arithmExprComma(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprAssign, Comma)
}

func (p *Parser) arithmExprAssign(compact bool) ArithmExpr {
	// Assign is different from the other binary operators because it's
	// right-associative and needs to check that it's placed after a name
	value := p.arithmExprTernary(compact)
	switch BinAritOperator(p.tok) {
	case AddAssgn, SubAssgn, MulAssgn, QuoAssgn, RemAssgn, AndAssgn,
		OrAssgn, XorAssgn, ShlAssgn, ShrAssgn, Assgn:
		if compact && p.spaced {
			return value
		}
		if !isArithName(value) {
			p.posErr(p.pos, "%s must follow a name", p.tok.String())
		}
		pos := p.pos
		tok := p.tok
		p.nextArithOp(compact)
		y := p.arithmExprAssign(compact)
		if y == nil {
			p.followErrExp(pos, tok.String())
		}
		return &BinaryArithm{
			OpPos: pos,
			Op:    BinAritOperator(tok),
			X:     value,
			Y:     y,
		}
	}
	return value
}

func (p *Parser) arithmExprTernary(compact bool) ArithmExpr {
	value := p.arithmExprLor(compact)
	if BinAritOperator(p.tok) != TernQuest || (compact && p.spaced) {
		return value
	}

	if value == nil {
		p.curErr("%s must follow an expression", p.tok.String())
	}
	questPos := p.pos
	p.nextArithOp(compact)
	if BinAritOperator(p.tok) == TernColon {
		p.followErrExp(questPos, TernQuest.String())
	}
	trueExpr := p.arithmExpr(compact)
	if trueExpr == nil {
		p.followErrExp(questPos, TernQuest.String())
	}
	if BinAritOperator(p.tok) != TernColon {
		p.posErr(questPos, "ternary operator missing : after ?")
	}
	colonPos := p.pos
	p.nextArithOp(compact)
	falseExpr := p.arithmExprTernary(compact)
	if falseExpr == nil {
		p.followErrExp(colonPos, TernColon.String())
	}
	return &BinaryArithm{
		OpPos: questPos,
		Op:    TernQuest,
		X:     value,
		Y: &BinaryArithm{
			OpPos: colonPos,
			Op:    TernColon,
			X:     trueExpr,
			Y:     falseExpr,
		},
	}
}

func (p *Parser) arithmExprLor(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprLand, OrArit)
}

func (p *Parser) arithmExprLand(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprBor, AndArit)
}

func (p *Parser) arithmExprBor(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprBxor, Or)
}

func (p *Parser) arithmExprBxor(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprBand, Xor)
}

func (p *Parser) arithmExprBand(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprEquality, And)
}

func (p *Parser) arithmExprEquality(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprComparison, Eql, Neq)
}

func (p *Parser) arithmExprComparison(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprShift, Lss, Gtr, Leq, Geq)
}

func (p *Parser) arithmExprShift(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprAddition, Shl, Shr)
}

func (p *Parser) arithmExprAddition(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprMultiplication, Add, Sub)
}

func (p *Parser) arithmExprMultiplication(compact bool) ArithmExpr {
	return p.arithmExprBinary(compact, p.arithmExprPower, Mul, Quo, Rem)
}

func (p *Parser) arithmExprPower(compact bool) ArithmExpr {
	// Power is different from the other binary operators because it's right-associative
	value := p.arithmExprUnary(compact)
	if BinAritOperator(p.tok) != Pow || (compact && p.spaced) {
		return value
	}

	if value == nil {
		p.curErr("%s must follow an expression", p.tok.String())
	}

	op := p.tok
	pos := p.pos
	p.nextArithOp(compact)
	y := p.arithmExprPower(compact)
	if y == nil {
		p.followErrExp(pos, op.String())
	}
	return &BinaryArithm{
		OpPos: pos,
		Op:    BinAritOperator(op),
		X:     value,
		Y:     y,
	}
}

func (p *Parser) arithmExprUnary(compact bool) ArithmExpr {
	if !compact {
		p.got(_Newl)
	}

	switch UnAritOperator(p.tok) {
	case Not, BitNegation, Plus, Minus:
		ue := &UnaryArithm{OpPos: p.pos, Op: UnAritOperator(p.tok)}
		p.nextArithOp(compact)
		if ue.X = p.arithmExprUnary(compact); ue.X == nil {
			p.followErrExp(ue.OpPos, ue.Op.String())
		}
		return ue
	}
	return p.arithmExprValue(compact)
}

func (p *Parser) arithmExprValue(compact bool) ArithmExpr {
	var x ArithmExpr
	switch p.tok {
	case addAdd, subSub:
		ue := &UnaryArithm{OpPos: p.pos, Op: UnAritOperator(p.tok)}
		p.nextArith(compact)
		if p.tok != _LitWord {
			p.followErr(ue.OpPos, token(ue.Op).String(), "a literal")
		}
		ue.X = p.arithmExprValue(compact)
		return ue
	case leftParen:
		pe := &ParenArithm{Lparen: p.pos}
		p.nextArithOp(compact)
		pe.X = p.followArithm(leftParen, pe.Lparen)
		pe.Rparen = p.matched(pe.Lparen, leftParen, rightParen)
		x = pe
	case leftBrack:
		p.curErr("[ must follow a name")
	case colon:
		p.curErr("ternary operator missing ? before :")
	case _LitWord:
		l := p.getLit()
		if p.tok != leftBrack {
			x = p.word(p.wps(l))
			break
		}
		pe := &ParamExp{Dollar: l.ValuePos, Short: true, Param: l}
		pe.Index = p.eitherIndex()
		x = p.word(p.wps(pe))
	case bckQuote:
		if p.quote == arithmExprLet && p.openBquotes > 0 {
			return nil
		}
		fallthrough
	default:
		if w := p.getWord(); w != nil {
			x = w
		} else {
			return nil
		}
	}

	if compact && p.spaced {
		return x
	}
	if !compact {
		p.got(_Newl)
	}

	// we want real nil, not (*Word)(nil) as that
	// sets the type to non-nil and then x != nil
	if p.tok == addAdd || p.tok == subSub {
		if !isArithName(x) {
			p.curErr("%s must follow a name", p.tok.String())
		}
		u := &UnaryArithm{
			Post:  true,
			OpPos: p.pos,
			Op:    UnAritOperator(p.tok),
			X:     x,
		}
		p.nextArith(compact)
		return u
	}
	return x
}

// nextArith consumes a token.
// It returns true if compact and the token was followed by spaces
func (p *Parser) nextArith(compact bool) bool {
	p.next()
	if compact && p.spaced {
		return true
	}
	if !compact {
		p.got(_Newl)
	}
	return false
}

func (p *Parser) nextArithOp(compact bool) {
	pos := p.pos
	tok := p.tok
	if p.nextArith(compact) {
		p.followErrExp(pos, tok.String())
	}
}

// arithmExprBinary is used for all left-associative binary operators
func (p *Parser) arithmExprBinary(compact bool, nextOp func(bool) ArithmExpr, operators ...BinAritOperator) ArithmExpr {
	value := nextOp(compact)
	for {
		var foundOp BinAritOperator
		for _, op := range operators {
			if p.tok == token(op) {
				foundOp = op
				break
			}
		}

		if token(foundOp) == illegalTok || (compact && p.spaced) {
			return value
		}

		if value == nil {
			p.curErr("%s must follow an expression", p.tok.String())
		}

		pos := p.pos
		p.nextArithOp(compact)
		y := nextOp(compact)
		if y == nil {
			p.followErrExp(pos, foundOp.String())
		}

		value = &BinaryArithm{
			OpPos: pos,
			Op:    foundOp,
			X:     value,
			Y:     y,
		}
	}
}

func isArithName(left ArithmExpr) bool {
	w, ok := left.(*Word)
	if !ok || len(w.Parts) != 1 {
		return false
	}
	switch x := w.Parts[0].(type) {
	case *Lit:
		return ValidName(x.Value)
	case *ParamExp:
		return x.nakedIndex()
	default:
		return false
	}
}

func (p *Parser) followArithm(ftok token, fpos Pos) ArithmExpr {
	x := p.arithmExpr(false)
	if x == nil {
		p.followErrExp(fpos, ftok.String())
	}
	return x
}

func (p *Parser) peekArithmEnd() bool {
	return p.tok == rightParen && p.r == ')'
}

func (p *Parser) arithmMatchingErr(pos Pos, left, right token) {
	switch p.tok {
	case _Lit, _LitWord:
		p.curErr("not a valid arithmetic operator: %s", p.val)
	case leftBrack:
		p.curErr("[ must follow a name")
	case colon:
		p.curErr("ternary operator missing ? before :")
	case rightParen, _EOF:
		p.matchingErr(pos, left, right)
	default:
		if p.quote == arithmExpr {
			p.curErr("not a valid arithmetic operator: %v", p.tok)
		}
		p.matchingErr(pos, left, right)
	}
}

func (p *Parser) matchedArithm(lpos Pos, left, right token) {
	if !p.got(right) {
		p.arithmMatchingErr(lpos, left, right)
	}
}

func (p *Parser) arithmEnd(ltok token, lpos Pos, old saveState) Pos {
	if !p.peekArithmEnd() {
		p.arithmMatchingErr(lpos, ltok, dblRightParen)
	}
	p.rune()
	p.postNested(old)
	pos := p.pos
	p.next()
	return pos
}
