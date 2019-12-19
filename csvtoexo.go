package main

import (
  "fmt"
  "os"
  "bufio"
  "regexp"
  "strings"
  "strconv"
  "encoding/csv"
  "encoding/hex"
  "unicode/utf8"

  "golang.org/x/text/encoding/japanese"
  "golang.org/x/text/transform"
  "golang.org/x/text/encoding/unicode"

  "./aviutlobj"
)

type myError struct {
  msg string
}
func (e myError) Error() string {
  return e.msg
}

const BUFSIZE = 1024

func loadExo(path string) ([]aviutlobj.AviUtlObject, error) {
  file, err := os.Open(path)
  if err != nil {
    return nil, myError{"exoファイルのパスが間違っています。: " + err.Error()}
  }
  defer file.Close()
  scanner := bufio.NewScanner(file)

  block := aviutlobj.NewBlock()
  aviutlObjects := make([]aviutlobj.AviUtlObject, 0)
  object := aviutlobj.NewAviUtlObject()
  for scanner.Scan() {
    line, _, err := transform.String(japanese.ShiftJIS.NewDecoder(), scanner.Text())
    if err != nil {
      return nil, myError{"ShiftJIS decofe error: " + err.Error()}
    }

    r := regexp.MustCompile(`\[(.+)\]`)
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

      var key string; var value string;
      for i, str := range slice {
        switch i {
          case 0:
            key = str
          case 1:
            value = str
        }
      }
      block = block.AppendMap(key, value)
      if key == "text" {
        hv, _ := hex.DecodeString(value)
        t, _, err := transform.Bytes(unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder(), hv)
        if err != nil {
          fmt.Println("UTF16LE error:", err.Error())
        }
        fmt.Println("after: ", string(t))
      }
    }
  }

  object.Blocks = append(object.Blocks, block)
  aviutlObjects = append(aviutlObjects, object)
  if scanner.Err() != nil {
    fmt.Println("Exo scan error")
    return nil, myError{"Exo scan error: " + err.Error()}
  }

  /*for _, obj := range aviutlObjects {
    fmt.Println("objGetName() : " + obj.GetName())
    for _, blk := range obj.Blocks {
      fmt.Println("blkName : " + blk.Name)
      for key, value := range blk.Params {
        fmt.Printf("prm : %s = %s\n", key, value)
      }
    }
  }*/
  return aviutlObjects, nil
}

func loadCsv(path string) ([][]string, error) {
  file, err := os.Open(path)
  if err != nil {
    return nil, myError{"csvファイルのパスが間違っています。: " + err.Error()}
  }
  defer file.Close()

  var csvStr string
  buf := make([]byte, BUFSIZE)
  for {
    n, err := file.Read(buf)
    if n == 0 {
      break
    }
    if err != nil {
      return nil, myError{"CSV file raad error: " + err.Error()}
    }

    csvStr += string(buf[:n])
  }

  r := csv.NewReader(strings.NewReader(csvStr))
  records, err := r.ReadAll()
  if err != nil {
    return nil, myError{"CSV string raad error: " + err.Error()}
  }

  isNoData := false
  if len(records) == 0 {
    isNoData = true
  } else if len(records[0]) == 0 {
    isNoData = true
  }
  if isNoData {
    return nil, myError{"CSV has no data."}
  }

  /*for _, record := range records {
    fmt.Printf("No.[%s] Team[%s] Artist[%s] Genre[%s] Title[%s]\n",
      record[0], record[1], record[2], record[3], record[4])
  }*/

  return records, nil
}

func makeExoStrFromCsv(objects []aviutlobj.AviUtlObject, records [][]string) (string, error) {
  newExoStr := objects[0].String()

  period := 1000
  objCount := 0
  for i := 0; i < len(records[0]); i++ {
    for j := 0; j < len(records); j++ {
      newObj := objects[i + 1]
      for index, _ := range newObj.Blocks {
        if index == 0 {
          newObj.Blocks[index].Name = strconv.Itoa(objCount)
        } else {
          newObj.Blocks[index].Name = strconv.Itoa(objCount) + "." + strconv.Itoa(index - 1)
        }
      }
      newObj.Blocks[0].Params["start"] = strconv.Itoa(i + period * j + 1)
      newObj.Blocks[0].Params["end"] = strconv.Itoa(i + period * (j + 1))
      newObj.Blocks[0].Params["group"] = strconv.Itoa(j + 1)

      t, _, err := transform.String(unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder(), records[j][i])
      if err != nil {
        return "", myError{"UTF-16LE encode error: " + err.Error()}
      }
      encoded := hex.EncodeToString([]byte(t))
      length := utf8.RuneCountInString(encoded)
      if length > 4096 - 4 {
        return "", myError{"Encoded text length is too long: " + strconv.Itoa(length)}
      }
      encoded += strings.Repeat("0", 4096 - length)
      newObj.Blocks[1].Params["text"] = encoded

      newExoStr += newObj.String()
      objCount++
    }
  }

  //fmt.Println(newExoStr)

  shiftJisExoStr, _, err := transform.String(japanese.ShiftJIS.NewEncoder(), newExoStr)
  if err != nil {
    return "", myError{"ShiftJIS encode error: " + err.Error()}
  }

  return shiftJisExoStr, nil
}

func createExo(str string) error {
  file, err := os.Create("./output.exo")
  if err != nil {
    return myError{"Exo file create error: " + err.Error()}
  }
  defer file.Close()

  file.Write(([]byte)(str))
  return nil
}

func main() {
  if len(os.Args) != 3 {
    fmt.Println("実行は csvtoexo <exo path> <csv path> で行ってください。")
    os.Exit(1)
  }

  objects, err := loadExo(os.Args[1])
  if err != nil {
    fmt.Println(err.Error())
    os.Exit(1)
  }

  records, err := loadCsv(os.Args[2])
  if err != nil {
    fmt.Println(err.Error())
    os.Exit(1)
  }

  objects = aviutlobj.DistinctLayer(objects)
  if len(objects) - 1 < len(records[0]) {
    fmt.Printf("元のExoに配置されているオブジェクトのレイヤーが足りません: layer=%d col=%d", len(objects) - 1, len(records[0]))
    os.Exit(1)
  }

  str, err := makeExoStrFromCsv(objects, records)
  if err != nil {
    fmt.Println(err.Error())
    os.Exit(1)
  }

  err = createExo(str)
  if err != nil {
    fmt.Println(err.Error())
    os.Exit(1)
  }
}
