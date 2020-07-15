package crawling
import "fmt"
import "strings"

type ErrorIdent struct {
  Type string
  Code int
  Idents []string
}

var ErrorMap = map[string]ErrorIdent{}

func init()  {
  ErrorMap["LocalAddrsOnly"] = ErrorIdent{
    Type:   "LocalAddrsOnly",
    Code:   400,
    Idents: []string{"has only local adresses"},
  }
  ErrorMap["AllDialsFailed"] = ErrorIdent{
    Type:   "AllDialsFailed",
    Code:   400,
    Idents: []string{"all dials failed"},
  }
  ErrorMap["ConnectionRefused"] = ErrorIdent{
    Type:   "ConnectionRefused",
    Code:   400,
    Idents: []string{"connect: connection refused"},
  }
  ErrorMap["SecurityProtocol"] = ErrorIdent{
    Type:   "SecurityProtocol",
    Code:   400,
    Idents: []string{"failed to negotiate security protocol"},
  }
}

type Status struct {

}

func IdentifyError(err error)  {
  errortext := strings.Split(err.Error(), "\n")
  for _, suberr := range errortext {
    fmt.Println(identifyLine(suberr))
  }

}

func identifyLine(line string) string {
  for errortype, errorinfo := range ErrorMap {
    for _, ident := range errorinfo.Idents {
      if strings.Contains(line, ident) {
        return errortype
      }
    }
  }
  return "Unknown"
}
