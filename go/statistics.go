package main

import (
	"fmt"
	"math"
	"sort"
)

func ComputeMaxDrawdown(prices []DailyPrice) (DrawdownResult, error) {
	if len(prices) < 2 {
		return DrawdownResult{}, fmt.Errorf("not enough prices to compute drawdown")
	}

	peak := prices[0]
	maxDrawdown := 0.0

	result := DrawdownResult{
		MaxDrawdown: 0.0,
		PeakDate:    prices[0].Date,
		PeakClose:   prices[0].Close,
		TroughDate:  prices[0].Date,
		TroughClose: prices[0].Close,
	}

	for _, p := range prices {
		if p.Close <= 0 {
			return DrawdownResult{}, fmt.Errorf("invalid close price on %s", FormatDate(p.Date))
		}

		if p.Close > peak.Close {
			peak = p
		}

		drawdown := (p.Close / peak.Close) - 1.0

		if drawdown < maxDrawdown {
			maxDrawdown = drawdown

			result = DrawdownResult{
				MaxDrawdown: drawdown,
				PeakDate:    peak.Date,
				PeakClose:   peak.Close,
				TroughDate:  p.Date,
				TroughClose: p.Close,
			}
		}
	}

	return result, nil
}

func ComputeDailyReturnStats(prices []DailyPrice) (DailyReturnStats, error) {
	if len(prices) < 2 {
		return DailyReturnStats{}, fmt.Errorf("not enough prices to compute daily returns")
	}

	returns := make([]float64, 0, len(prices)-1)

	stats := DailyReturnStats{
		WorstReturn: 0.0,
		BestReturn:  0.0,
	}

	for i := 1; i < len(prices); i++ {
		prev := prices[i-1]
		curr := prices[i]

		if prev.Close <= 0 || curr.Close <= 0 {
			return DailyReturnStats{}, fmt.Errorf("invalid close price near %s", FormatDate(curr.Date))
		}

		r := (curr.Close / prev.Close) - 1.0
		returns = append(returns, r)

		if i == 1 || r < stats.WorstReturn {
			stats.WorstReturn = r
			stats.WorstDate = curr.Date
		}

		if i == 1 || r > stats.BestReturn {
			stats.BestReturn = r
			stats.BestDate = curr.Date
		}
	}

	stats.Count = len(returns)
	stats.Mean = meanFloat64(returns)
	stats.Median = medianFloat64(returns)
	stats.AnnualizedVolatility = sampleStdDevFloat64(returns) * math.Sqrt(252.0)

	return stats, nil
}

func meanFloat64(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}

	var sum float64
	for _, x := range xs {
		sum += x
	}

	return sum / float64(len(xs))
}

func medianFloat64(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}

	ys := append([]float64(nil), xs...)
	sort.Float64s(ys)

	n := len(ys)
	if n%2 == 1 {
		return ys[n/2]
	}

	return (ys[n/2-1] + ys[n/2]) / 2.0
}

func sampleStdDevFloat64(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}

	mean := meanFloat64(xs)

	var sumSquares float64
	for _, x := range xs {
		diff := x - mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(xs)-1)
	return math.Sqrt(variance)
}
