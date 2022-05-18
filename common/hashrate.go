package common

import (
	"fmt"
	"github.com/shopspring/decimal"
)

type HashRate float64

var (
	HashRateUnit  = int64(1)
	HashRateUnitK = HashRateUnit * int64(1000)
	HashRateUnitM = HashRateUnitK * int64(1000)
	HashRateUnitG = HashRateUnitM * int64(1000)
	HashRateUnitT = HashRateUnitG * int64(1000)
	HashRateUnitP = HashRateUnitT * int64(1000)
	HashRateUnitE = HashRateUnitP * int64(1000)
)

func (hr HashRate) K() float64 {
	var nhr = float64(hr)
	decimal.DivisionPrecision = 2
	n := decimal.NewFromFloat(nhr)
	v := n.Div(decimal.NewFromInt(HashRateUnitK))
	m := v.Mod(decimal.NewFromInt(HashRateUnitK))
	d := m.Div(decimal.NewFromInt(HashRateUnitK))
	return v.Add(d).InexactFloat64()
}

func (hr HashRate) M() float64 {
	var nhr = float64(hr)
	decimal.DivisionPrecision = 2
	n := decimal.NewFromFloat(nhr)
	v := n.Div(decimal.NewFromInt(HashRateUnitM))
	m := v.Mod(decimal.NewFromInt(HashRateUnitM))
	d := m.Div(decimal.NewFromInt(HashRateUnitM))
	return v.Add(d).InexactFloat64()
}
func (hr HashRate) G() float64 {
	var nhr = float64(hr)
	decimal.DivisionPrecision = 2
	n := decimal.NewFromFloat(nhr)
	v := n.Div(decimal.NewFromInt(HashRateUnitG))
	m := v.Mod(decimal.NewFromInt(HashRateUnitG))
	d := m.Div(decimal.NewFromInt(HashRateUnitG))
	return v.Add(d).InexactFloat64()
}

func (hr HashRate) T() float64 {
	var nhr = float64(hr)
	decimal.DivisionPrecision = 2
	n := decimal.NewFromFloat(nhr)
	v := n.Div(decimal.NewFromInt(HashRateUnitT))
	m := v.Mod(decimal.NewFromInt(HashRateUnitT))
	d := m.Div(decimal.NewFromInt(HashRateUnitT))
	return v.Add(d).InexactFloat64()
}
func (hr HashRate) P() float64 {
	var nhr = float64(hr)
	decimal.DivisionPrecision = 2
	n := decimal.NewFromFloat(nhr)
	v := n.Div(decimal.NewFromInt(HashRateUnitP))
	m := v.Mod(decimal.NewFromInt(HashRateUnitP))
	d := m.Div(decimal.NewFromInt(HashRateUnitP))
	return v.Add(d).InexactFloat64()
}
func (hr HashRate) E() float64 {
	var nhr = float64(hr)
	decimal.DivisionPrecision = 2
	n := decimal.NewFromFloat(nhr)
	v := n.Div(decimal.NewFromInt(HashRateUnitE))
	m := v.Mod(decimal.NewFromInt(HashRateUnitE))
	d := m.Div(decimal.NewFromInt(HashRateUnitE))
	return v.Add(d).InexactFloat64()
}
func (hr HashRate) ToString(suffix string) string {
	if suffix == "" {
		suffix = "s"
	}
	var nhr = float64(hr)
	var s = fmt.Sprintf("%.2f Hash/%s", hr, suffix)
	if nhr >= float64(HashRateUnitK) && nhr < float64(HashRateUnitM) {
		s = fmt.Sprintf("%.2f KHash/%s", hr.K(), suffix)
	} else if nhr >= float64(HashRateUnitM) && nhr < float64(HashRateUnitG) {
		s = fmt.Sprintf("%.2f MHash/%s", hr.M(), suffix)
	} else if nhr >= float64(HashRateUnitG) && nhr < float64(HashRateUnitT) {
		s = fmt.Sprintf("%.2f GHash/%s", hr.G(), suffix)
	} else if nhr >= float64(HashRateUnitT) && nhr < float64(HashRateUnitP) {
		s = fmt.Sprintf("%.2f THash/%s", hr.T(), suffix)
	} else if nhr >= float64(HashRateUnitP) && nhr < float64(HashRateUnitE) {
		s = fmt.Sprintf("%.2f PHash/%s", hr.P(), suffix)
	} else if nhr >= float64(HashRateUnitE) {
		s = fmt.Sprintf("%.2f EHash/%s", hr.E(), suffix)
	}
	return s
}

func (hr HashRate) String() string {
	return hr.ToString("s")
}
