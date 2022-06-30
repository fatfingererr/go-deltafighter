package main

import (
	"fmt"
	"math"

	"github.com/takuoki/clmconv"
	"github.com/xuri/excelize/v2"
)

func getMinMax(data KLineData) (minPrice float64, maxPrice float64) {
	max := data.highs[0]
	min := data.lows[0]
	for i, _ := range data.opentimes {
		if max < data.highs[i] {
			max = data.highs[i]
		}
		if min > data.lows[i] {
			min = data.lows[i]
		}
	}
	return min, max
}

func getCell(row int, col int) (location string) {
	return fmt.Sprintf("%s%d", clmconv.Itoa(row-1), col)
}

func main() {
	const unitTimeframe = "1m"
	const timeframe = "15m"
	const symbol = "BTCUSDT"

	tickSize := GetSymbolTickSize(symbol)
	pRange := 1000 * tickSize
	unitData := GetKLineData(symbol, unitTimeframe)
	data := GetKLineData(symbol, timeframe)
	minPrice, maxPrice := getMinMax(unitData)
	iLow := int(math.Floor(minPrice / pRange))
	iHigh := int(math.Ceil(maxPrice / pRange))
	var pLow, pHigh []float64
	for i := iLow; i <= iHigh; i++ {
		pLow = append(pLow, float64(i)*pRange)
		pHigh = append(pHigh, float64(i+1)*pRange)
	}
	buyer := make([][]float64, 0)
	seller := make([][]float64, 0)
	for i := range pLow {
		bcol := make([]float64, 0)
		scol := make([]float64, 0)
		for j := range data.opens {
			bcol = append(bcol, 0)
			scol = append(scol, 0)
			_ = i + j
		}
		buyer = append(buyer, bcol)
		seller = append(seller, scol)
	}

	nowtime := 0
	for di := range unitData.opens {
		uHigh := int(math.Floor(unitData.highs[di] / pRange))
		uLow := int(math.Floor(unitData.lows[di] / pRange))
		uRange := uHigh - uLow + 1
		fmt.Printf("DI!! nowtick: %s \n", unitData.opentimes[di].String())
		for ri := iLow; ri < iHigh; ri++ {
			if uLow >= ri && uLow < ri+1 {
				i := ri - iLow
				// fmt.Printf("uLow: %d, ri: %d, i: %d, pLow: %f, low: %f \n", uLow, ri, i, pLow[i], unitData.lows[di])
				for j := nowtime; j < len(data.opens); j++ {
					tstart := data.opentimes[j].Unix()
					tend := data.closetimes[j].Unix()
					tnow := unitData.opentimes[di].Unix()
					fmt.Printf("bar open: %s, bar closed: %s, nowtick: %s \n", data.opentimes[j].String(), data.closetimes[j].String(), unitData.opentimes[di].String())
					if tnow >= tstart && tnow <= tend {
						fmt.Printf("!")
						bSize := unitData.takers[di] / float64(uRange)
						aSize := (unitData.volumes[di] - unitData.takers[di]) / float64(uRange)
						fmt.Printf("unitData.volumes[di]: %f, unitData.takers[di]: %f, unitData.makers[di]: %f\n", unitData.volumes[di], unitData.takers[di], unitData.makers[di])
						for ii := i; ii < i+uRange+1; ii++ {
							fmt.Printf("buyer: %f, seller: %f\n", buyer[ii][j], seller[ii][j])
							fmt.Printf("bSize: %f, aSize: %f\n", bSize, aSize)
							buyer[ii][j] = buyer[ii][j] + bSize
							seller[ii][j] = seller[ii][j] + aSize
							fmt.Printf("buyer(%d,%d): %f, seller(%d,%d): %f\n", ii, j, buyer[ii][j], ii, j, seller[ii][j])
						}
						break
					} else {
						nowtime += 1
					}
				}
				break
			}
		}
	}

	sheet := "Delta"
	f := excelize.NewFile()
	index := f.NewSheet(sheet)
	count := 0

	for i := range pLow {
		headerStyle, _ := f.NewStyle(&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "left"},
		})
		f.SetCellValue(sheet, getCell(1, i+3), fmt.Sprintf("%.2f", pLow[len(pLow)-i-1]))
		f.SetCellStyle(sheet, getCell(1, i+3), getCell(1, 1+3), headerStyle)
	}

	for j := range data.opens {
		isCount := false
		rshift := 3
		headerStyle, _ := f.NewStyle(&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "center"},
		})
		f.SetCellValue(sheet, getCell(count+2, 1), fmt.Sprintf("%02d/%02d", data.opentimes[j].Month(), data.opentimes[j].Day()))
		f.SetCellValue(sheet, getCell(count+2, 2), fmt.Sprintf("%02d:%02d", data.opentimes[j].Hour(), data.opentimes[j].Minute()))
		f.SetCellStyle(sheet, getCell(count+2, 1), getCell(count+2, 2), headerStyle)

		for i := range pLow {
			invi := len(pLow) - i - 1
			if buyer[i][j] != 0 && seller[i][j] != 0 {
				isCount = true
				f.SetCellValue(sheet, getCell(count+2, invi+rshift), fmt.Sprintf("%.0f x %.0f", buyer[i][j], seller[i][j])) // fmt.Sprintf("%.2f x %.2f", buyer[i][j], seller[i][j])
				var vred, vblue int
				buyerRatio := buyer[i][j]/seller[i][j] - 1
				if buyerRatio >= 0 {
					vblue = 0
					vred = int(math.Max(0, math.Min(255, 200*math.Pow(buyer[i][j]/seller[i][j]-1, 2))))
				} else {
					vred = 0
					vblue = int(math.Max(0, math.Min(255, 200*math.Pow(seller[i][j]/buyer[i][j]-1, 2))))
				}

				red := fmt.Sprintf("#%02X%02X%02X", vred, 0, vblue)

				color := []string{"#FFFFFF", red}
				// fmt.Print(int(math.Max(0, math.Min(255, 127+100*(buyer[i][j]/seller[i][j]-1)))))

				style, err := f.NewStyle(&excelize.Style{
					Fill:      excelize.Fill{Type: "gradient", Color: color, Shading: 1},
					Alignment: &excelize.Alignment{Horizontal: "center"},
				})
				if err != nil {
					fmt.Println(err)
					return
				}
				f.SetCellStyle(sheet, getCell(count+2, invi+rshift), getCell(count+2, invi+rshift), style)
			} else {
				f.SetCellValue(sheet, getCell(count+2, invi+rshift), "")
			}
		}
		if isCount {
			count += 1
		}
	}
	f.SetColWidth(sheet, clmconv.Itoa(0), clmconv.Itoa(count), 13)
	f.SetActiveSheet(index)
	f.SetPanes(sheet, `{"freeze":true,"x_split":1,"y_split":2,"top_left_cell":"B3","active_pane":"bottomRight","panes":[{"pane":"topLeft"},{"pane":"topRight"},{"pane":"bottomLeft"},{"active_cell":"B3", "sqref":"B3", "pane":"bottomRight"}]}`)

	if err := f.SaveAs(fmt.Sprintf("%s_%s_%d.xlsx", symbol, timeframe, unitData.opentimes[len(unitData.opentimes)-1].Unix())); err != nil {
		fmt.Println(err)
	}

	// fmt.Printf("price range : %f", pRange)
	// fmt.Printf("minPrice : %f", minPrice)
	// fmt.Print(buyer[0])
	// fmt.Printf("len : %d", len(pLow))
}
