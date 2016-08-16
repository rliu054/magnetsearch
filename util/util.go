package util

import "math/big"

// const (
// 	NODENUM = 100
// )

// Mid returns (x+y)/2.
func Mid(x, y *big.Int) *big.Int {
	sum := big.NewInt(0).Add(x, y)
	return sum.Rsh(sum, uint(1))
}

func Biadd(x, y *big.Int) *big.Int {
	return big.NewInt(0).Add(x, y)
}

func Binew(x int64) *big.Int {
	return big.NewInt(x)
}

// func id2bi(id Identifier) *big.Int {
// 	return big.NewInt(0).SetBytes(id)
// }

func Birsh(x int64, n uint) *big.Int {
	return big.NewInt(0).Rsh(Binew(x), n)
}

func Bilsh(x int64, n uint) *big.Int {
	return big.NewInt(0).Lsh(Binew(x), n)
}

func Bisub(x, y *big.Int) *big.Int {
	return big.NewInt(0).Sub(x, y)
}

func Bidiv(x, y *big.Int) *big.Int {
	return big.NewInt(0).Div(x, y)
}

func Bimid(max, min *big.Int) *big.Int {
	d := Bisub(max, min)
	return d.Rsh(d, 1).Add(d, min)
}
