package main

import (
	"bufio"
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"github.com/Shimi9999/csvtoexo/aviutlobj"
)

func main() {
	var (
		duration = flag.String("duration", "", "duration exo")
	)
	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Println("Usage: csvtoexo [-duration <duration exopath>] <template exopath> <csvpath>")
		os.Exit(1)
	}

	templateExoPath := flag.Arg(0)
	csvpath := flag.Arg(1)

	objects, err := loadExo(templateExoPath)
	if err != nil {
		fmt.Println("loadExo error:", err.Error())
		os.Exit(1)
	}

	records, err := loadCsv(csvpath)
	if err != nil {
		fmt.Println("loadCsv error:", err.Error())
		os.Exit(1)
	}

	objects = aviutlobj.DistinctLayer(objects)
	if len(objects)-1 < len(records[0]) {
		fmt.Printf("テンプレートexoファイルに配置されているオブジェクト数がcsvデータのカラム数より少ないです: layer=%d, column=%d",
			len(objects)-1, len(records[0]))
		os.Exit(1)
	}

	var str string
	if *duration != "" {
		durObjects, err := loadExo(*duration)
		if err != nil {
			fmt.Println("Duration loadExo error:", err.Error())
			os.Exit(1)
		}
		if len(durObjects) == 0 {
			fmt.Println("Duration exo has no objects.")
			os.Exit(1)
		}

		str, err = makeExoStrFromCsv(objects, durObjects, records)
		if err != nil {
			fmt.Println("makeExoStrFromCsv error:", err.Error())
			os.Exit(1)
		}
	} else {
		str, err = makeExoStrFromCsv(objects, nil, records)
		if err != nil {
			fmt.Println("makeExoStrFromCsv error:", err.Error())
			os.Exit(1)
		}
	}

	err = createExo(str)
	if err != nil {
		fmt.Println("createExo error:", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Finish: output.exo generated.")
}

func loadExo(path string) ([]aviutlobj.AviUtlObject, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("exo Open: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	block := aviutlobj.NewBlock()
	aviutlObjects := make([]aviutlobj.AviUtlObject, 0)
	object := aviutlobj.NewAviUtlObject()
	for scanner.Scan() {
		line, _, err := transform.String(japanese.ShiftJIS.NewDecoder(), scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("ShiftJIS decode: %v", err)
		}

		r := regexp.MustCompile(`^\[.+\]$`)
		if r.MatchString(line) {
			name := r.FindString(line)
			name = strings.Trim(strings.Trim(name, "["), "]")

			if block.Name != "" {
				object.Blocks = append(object.Blocks, block)
				block = aviutlobj.NewBlock()

				if !strings.Contains(name, ".") {
					aviutlObjects = append(aviutlObjects, object)
					object = aviutlobj.NewAviUtlObject()
				}
			}
			block.Name = name
		} else {
			slice := strings.Split(line, "=")

			var key string
			var value string
			for i, str := range slice {
				switch i {
				case 0:
					key = str
				case 1:
					value = str
				}
			}
			block = block.AppendMap(key, value)
		}
	}

	object.Blocks = append(object.Blocks, block)
	aviutlObjects = append(aviutlObjects, object)
	if scanner.Err() != nil {
		return nil, fmt.Errorf("exo Scan: %v", scanner.Err())
	}

	return aviutlObjects, nil
}

func loadCsv(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("csv Open: %v", err)
	}
	defer file.Close()

	r := csv.NewReader(file)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv ReadAll: %v", err)
	}

	isNoData := false
	if len(records) == 0 {
		isNoData = true
	} else if len(records[0]) == 0 {
		isNoData = true
	}
	if isNoData {
		return nil, fmt.Errorf("csv has no data.")
	}

	return records, nil
}

func makeExoStrFromCsv(objects []aviutlobj.AviUtlObject, durationObjs []aviutlobj.AviUtlObject, records [][]string) (string, error) {
	newExoStr := objects[0].String()

	period := 500
	objCount := 0
	for i := 0; i < len(records[0]); i++ {
		var beforeObjEnd string
		for j := 0; j < len(records); j++ {
			newObj := objects[i+1]
			for index := range newObj.Blocks {
				if index == 0 {
					newObj.Blocks[index].Name = strconv.Itoa(objCount)
				} else {
					newObj.Blocks[index].Name = strconv.Itoa(objCount) + "." + strconv.Itoa(index-1)
				}
			}
			if durationObjs != nil {
				if len(durationObjs) > j+1 { // durationObjs[0] = [exedit] object
					newObj.Blocks[0].Params["start"] = durationObjs[j+1].Blocks[0].Params["start"]
					newObj.Blocks[0].Params["end"] = durationObjs[j+1].Blocks[0].Params["end"]
				} else {
					end, _ := strconv.Atoi(beforeObjEnd)
					newObj.Blocks[0].Params["start"] = strconv.Itoa(end + 1)
					newObj.Blocks[0].Params["end"] = strconv.Itoa(end + period)
				}
			} else {
				newObj.Blocks[0].Params["start"] = strconv.Itoa(period*j + 1)
				newObj.Blocks[0].Params["end"] = strconv.Itoa(period * (j + 1))
			}
			if newObj.Blocks[0].Params["start"] == "" {
				fmt.Println("start is empty:", newObj.Blocks[0].Name, j)
			}
			beforeObjEnd = newObj.Blocks[0].Params["end"]
			newObj.Blocks[0].Params["group"] = strconv.Itoa(j + 1)

			t, _, err := transform.String(unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder(),
				strings.Replace(records[j][i], "\n", "\r\n", -1))
			if err != nil {
				return "", fmt.Errorf("UTF-16LE encode: %v", err)
			}
			encoded := hex.EncodeToString([]byte(t))
			length := utf8.RuneCountInString(encoded)
			if length > 4096-4 {
				return "", fmt.Errorf("Encoded text length is too long: %s", strconv.Itoa(length))
			}
			encoded += strings.Repeat("0", 4096-length)
			newObj.Blocks[1].Params["text"] = encoded

			newExoStr += newObj.String()
			objCount++
		}
	}

	shiftJisExoStr, _, err := transform.String(japanese.ShiftJIS.NewEncoder(), newExoStr)
	if err != nil {
		return "", fmt.Errorf("ShiftJIS encode: %v", err)
	}

	return shiftJisExoStr, nil
}

func createExo(str string) error {
	file, err := os.Create("./output.exo")
	if err != nil {
		return fmt.Errorf("exo Create: %v", err)
	}
	defer file.Close()

	_, err = file.Write(([]byte)(str))
	if err != nil {
		return fmt.Errorf("exo White: %v", err)
	}

	return nil
}
