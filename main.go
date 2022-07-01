package main

import (
	"fmt"
	"github.com/takuoki/clmconv"
	"github.com/xuri/excelize/v2"
	"math"
	"net/http"
	"time"

	"html/template"
	"log"
	// "strings"
	"path"
)

func getMinMax(data KLineData) (minPrice float64, maxPrice float64) {
	max := data.highs[0]
	min := data.lows[0]
	for i := range data.opentimes {
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

func generateExcel(){

	const unitTimeframe = "1m"
	const timeframe = "15m"
	const symbol = "ETHUSDT"

	tickSize := GetSymbolTickSize(symbol)
	pRange := 500 * tickSize
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
}


func CalcDeltaData(unitTimeframe string,timeframe string,symbol string) *DetalViewModel{

	var viewModel DetalViewModel
	var datas = map[string]DeltaData{}
	viewModel.Timeframe=timeframe
	viewModel.UnitTimeframe=timeframe
	viewModel.Symbol = symbol

	tickSize := GetSymbolTickSize(symbol)
	pRange := 500 * tickSize
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

	viewModel.Times = data.opentimes

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

							deltaData := DeltaData{
								Price:      pLow[ii],
								Time:        data.opentimes[j],
								MarketSell: seller[ii][j],
								MarketBuy:  buyer[ii][j],
							}

							buyerRatio := deltaData.MarketBuy/deltaData.MarketSell - 1

							var vblue = 0
							var vred = 0
							if buyerRatio >= 0 {
								vblue = 0
								vred = int(math.Max(0, math.Min(255, 200*math.Pow(deltaData.MarketBuy/deltaData.MarketSell - 1, 2))))
							} else {
								vred = 0
								vblue = int(math.Max(0, math.Min(255, 200*math.Pow(deltaData.MarketSell/deltaData.MarketBuy - 1, 2))))
							}

							color := fmt.Sprintf("#%02X%02X%02X", vred, 0, vblue)

							deltaData.Color = color

							datas[fmt.Sprintf("%d%d%d%d-%f",deltaData.Time.Month(),deltaData.Time.Day(),deltaData.Time.Hour(),deltaData.Time.Minute(),deltaData.Price)] = deltaData
							//datas = append(datas , deltaData)
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

	var reversePrices  []float64
	for i := range pLow{
		reversePrices = append(reversePrices, pLow[len(pLow)-i -1])
	}

	viewModel.Prices = reversePrices

	//for i := range pLow {
	//	headerStyle, _ := f.NewStyle(&excelize.Style{
	//		Alignment: &excelize.Alignment{Horizontal: "left"},
	//	})
	//	f.SetCellValue(sheet, getCell(1, i+3), fmt.Sprintf("%.2f", pLow[len(pLow)-i-1]))
	//	f.SetCellStyle(sheet, getCell(1, i+3), getCell(1, 1+3), headerStyle)
	//}
	viewModel.Datas = datas
	return &viewModel
}


type DeltaData struct {
	Time       time.Time
	Price      float64
	Interval   string
	MarketBuy  float64
	MarketSell float64
	Color      string
}

type DetalViewModel struct {
	Datas map[string]DeltaData
	Times []time.Time
	Prices []float64
	UnitTimeframe string
	Timeframe string
	Symbol string
}

func renderWeb(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()  //解析參數，預設是不會解析的
    // fmt.Println(r.Form)  //這些資訊是輸出到伺服器端的列印資訊
    // fmt.Println("path", r.URL.Path)
    // fmt.Println("scheme", r.URL.Scheme)
    // fmt.Println(r.Form["url_long"])
    // for k, v := range r.Form {
    //     fmt.Println("key:", k)
    //     fmt.Println("val:", strings.Join(v, ""))
    // }
    // fmt.Fprintf(w, "Hello astaxie!") //這個寫入到 w 的是輸出到客戶端的

	var unitTimeframe = "1m"
	var timeframe = "15m"
	if len(r.Form["timeframe"]) > 0 {
		timeframe = r.Form["timeframe"][0]
	}
	var symbol = "ETHUSDT"
	if len(r.Form["symbol"]) > 0 {
		symbol = r.Form["symbol"][0]
	}

	if len(timeframe) == 0{
		timeframe = "15m"
	}

	if len(symbol) == 0{
		symbol = "ETHUSDT"
	}

	var model = CalcDeltaData(unitTimeframe,timeframe,symbol)

	fm := template.FuncMap{
		"ttime": func(t time.Time) string {
			return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
		},
		"tdate": func(t time.Time) string {
			return 		fmt.Sprintf("%02d/%02d", t.Month(), t.Day())
		},
		"lookup": func(t time.Time,price float64, data map[string]DeltaData) string{
			val,exist := data[fmt.Sprintf("%d%d%d%d-%f",t.Month(),t.Day(),t.Hour(),t.Minute(),price)]
			if !exist {
				return ""
			}
			return fmt.Sprintf("%.0f x %.0f",val.MarketBuy,val.MarketSell)
		},
		"lookupColor": func(t time.Time,price float64, data map[string]DeltaData) string{
			val,exist := data[fmt.Sprintf("%d%d%d%d-%f",t.Month(),t.Day(),t.Hour(),t.Minute(),price)]
			if !exist {
				return ""
			}
			return val.Color
		},
	}

    fp := path.Join("view", "index.html")


    tmpl, err := template.New("index.html").Funcs(fm).ParseFiles(fp)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    if err := tmpl.Execute(w, model); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }

}

func main() {
    http.HandleFunc("/", renderWeb) //設定存取的路由
    err := http.ListenAndServe(":9090", nil) //設定監聽的埠
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }

	// fmt.Printf("price range : %f", pRange)
	// fmt.Printf("minPrice : %f", minPrice)
	// fmt.Print(buyer[0])
	// fmt.Printf("len : %d", len(pLow))
}
