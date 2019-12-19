package aviutlobj

import (
  "fmt"
  "strings"
)

type Block struct {
  Name string
  Params map[string]string
  KeyOrder []string
}

func NewBlock() Block {
  var block Block
  block.Params = make(map[string]string)
  block.KeyOrder = make([]string, 0)
  return block
}

func (block Block) AppendMap(key string, value string) Block {
  block.Params[key] = value
  block.KeyOrder = append(block.KeyOrder, key)
  return block
}

type AviUtlObject struct {
  Blocks []Block
}

func NewAviUtlObject() AviUtlObject {
  var obj AviUtlObject
  obj.Blocks = make([]Block, 0)
  return obj
}

func (obj AviUtlObject) GetName() string {
  if len(obj.Blocks) > 0 {
    return obj.Blocks[0].Name
  }
  return ""
}

func (obj AviUtlObject) String() string {
  var str string
  for _, block := range obj.Blocks {
    str += "[" + block.Name + "]\n"
    for _, key := range block.KeyOrder {
      str += key + "=" + block.Params[key] + "\n"
    }
  }
  return str
}

func containLayer(layers []string, target string) bool {
  for _, l := range layers {
    if l == target {
      return true
    }
  }
  return false
}

func removeObject(objects []AviUtlObject, targetIndex int) []AviUtlObject {
  result := []AviUtlObject{}
  for index, obj := range objects {
    if index != targetIndex {
      result = append(result, obj)
    }
  }
  fmt.Printf("removed!: %d\n", targetIndex)
  return result
}

func DistinctLayer(objs []AviUtlObject) []AviUtlObject {
  var layers []string
  for i := 0; i < len(objs); i++ {
    for _, block := range objs[i].Blocks {
      if !strings.Contains(block.Name, ".") && block.Name != "exedit" {
        if containLayer(layers, block.Params["layer"]) {
          objs = removeObject(objs, i)
          i--
        } else {
          layers = append(layers, block.Params["layer"])
        }
        break
      }
    }
  }
  return objs
}
