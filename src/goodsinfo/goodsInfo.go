package main

import(
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb  "github.com/hyperledger/fabric/protos/peer"
	"strconv"
)

type goodsinfo struct {
	Identity string      `json:"identity"`  //货物编号
	Num      int         `json:"num"`       //数量
	State    bool		 `json:"state"`		//状态true还有库存 false无库存
	Name     string      `json:"name"`     //名称
	Msp		 string		 `json:"msp"`		//管理员身份 user为空
}

type historyinfo struct {
	Goodsinfos []goodsinfo `json:"goodsinfos"`
}

func (g *goodsinfo)  Init(shim.ChaincodeStubInterface) pb.Response{
	return shim.Success(nil)
}

func (g *goodsinfo) Invoke(stub shim.ChaincodeStubInterface) pb.Response  {
	funcName,args := stub.GetFunctionAndParameters()
	if funcName =="create" {				//保存货物信息
		return g.createGoods(stub,args)
	}else if funcName =="save" {
		return g.goodsave(stub,args) //修改货物信息 #执行前需判断货物数量
	}else if funcName =="query" {
		return g.goodsquery(stub,args) //查询货物信息
	}else if funcName == "delete" {
		return g.delete(stub,args) //删除货物信息 #执行前需判断货物数量
	}else if funcName == "queryhistory" {
		return g.queryhistory(stub,args) //查询所有历史记录
	}else if funcName == "querymark" {
		return g.getQueryResult(stub,args) //实现分页查询
	}else{
		return shim.Error("no match function")
	}
}

func (g *goodsinfo) goodsquery(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("except one arg")
	}else {
		value,err := stub.GetState(args[0])
		if err!=nil {
			return shim.Error("no data found")
		}
		return shim.Success(value)
	}
}


//@args[0] 查询项目名称:查询对象名称
//@args[1] 查询单次数量
//@args[2] 查询的页面hash值 第一次调用可以直接输入空值
//返回对象json数组+书签hash值 书签用于下次查询
func (g *goodsinfo) findall(stub shim.ChaincodeStubInterface, args []string) pb.Response{
	if len(args)!=3 {
		return shim.Error("no except three args findname name num bookmark")
	}
	var queryString = "{\"selector\":"+args[0]+"}"
	var bookmark = args[2]
	var pagesize,err = strconv.ParseInt(args[1],10,32)
	if err != nil {
		return shim.Error(err.Error())
	}
	queryResults,responseMetadata, err := stub.GetQueryResultWithPagination(queryString,int32(pagesize),bookmark) //必须是CouchDB才行
	if err!=nil{
		return shim.Error("query failed")
	}
	var Goodsinfo []goodsinfo
	for queryResults.HasNext() {
		var keym,err = queryResults.Next()
		if err!= nil {
			return shim.Error(err.Error())
		}
		value := keym.Value
		goodsInfo := new(goodsinfo)
		var er = json.Unmarshal(value, goodsInfo)
		if er!=nil {
			return shim.Error(er.Error())
		}
		Goodsinfo = append(Goodsinfo, *goodsInfo)
	}
	var historyInfo = new(historyinfo)
	historyInfo.Goodsinfos = Goodsinfo
	var val, err2 = json.Marshal(historyInfo)
	if err2 != nil {
		return shim.Error(err2.Error())
	} else {
		var v,e =json.Marshal(responseMetadata.Bookmark)
		if e!=nil {
			return shim.Error(e.Error())
		}else {
			val = append(val, v...)
			return shim.Success(val)
		}
	}
}

func (g *goodsinfo) goodsave(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=2 {
		return shim.Error("except two args")
	}else {
		var val, err2 = stub.GetState(args[0])
		if err2 !=nil {
			return shim.Error(err2.Error())
		}else if val == nil || string(val) == "{\"identity\":\"unkown\"}" {
			return shim.Error("key never create")
		}
		var err=stub.PutState(args[0],[]byte(args[1]))
		if err!=nil {
			return shim.Error(err.Error())
		}else {
			return shim.Success([]byte("Success"))
		}
	}
}

func (g *goodsinfo) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=1 {
		return shim.Error("except one arg")
	}else {
		var value,err = stub.GetState(args[0])
		var check = string(value)
		if check == "{\"identity\":\"unkown\"}" {
			return shim.Error("data already delete")
		}
		if err!=nil || value==nil {
			return shim.Error("data delete fail")
		}else {
			err = stub.PutState(args[0],[]byte("{\"identity\":\"unkown\"}"))
			return shim.Success([]byte("Success"))
		}
	}
}

func (g *goodsinfo) queryhistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=1 {
		return shim.Error("except one arg")
	}else {
		var goodsinfos []goodsinfo
		it,err := stub.GetHistoryForKey(args[0])
		if err!=nil {
			return shim.Error("no data find")
		}else {
			for it.HasNext() {
				keym,err :=it.Next()
				if err!=nil {
					return shim.Error("data get error")
				}
				value :=keym.Value
				basicInfo := new(goodsinfo)
				er :=json.Unmarshal(value,basicInfo)
				if er != nil {
					return shim.Error(er.Error())
				}
				goodsinfos = append(goodsinfos, *basicInfo)
			}
			historyInfo:= new(historyinfo)
			historyInfo.Goodsinfos = goodsinfos
			value,err :=json.Marshal(historyInfo)
			if err!=nil {
				return shim.Error(err.Error())
			}else {
				return shim.Success(value)
			}
		}
	}
}

func (g *goodsinfo) createGoods(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=2 {
		return shim.Error("except two args")
	}
	var val, err = stub.GetState(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}else if val==nil  {
		err = stub.PutState(args[0], []byte(args[1]))
		if err !=nil {
			return shim.Error(err.Error())
		}else {
			return shim.Success([]byte("Success"))
		}
	}
	return shim.Error("key already exists")
}


//@args[0] 查询项目名称:查询对象名称
//@args[1] 查询单次数量
//@args[2] 查询的页面hash值 第一次调用可以直接输入空值
//返回对象json数组+书签hash值 书签用于下次查询
func (g *goodsinfo) getQueryResult(stub shim.ChaincodeStubInterface, args []string) pb.Response{
	if len(args)!=3 {
		return shim.Error("no except three args findname name num bookmark")
	}
	var queryString = "{\"selector\":"+args[0]+"}"
	var bookmark = args[2]
	var pagesize,err = strconv.ParseInt(args[1],10,32)
	if err != nil {
		return shim.Error(err.Error())
	}
	queryResults,responseMetadata, err := stub.GetQueryResultWithPagination(queryString,int32(pagesize),bookmark) //必须是CouchDB才行
	if err!=nil{
		return shim.Error("query failed")
	}
	var Goodsinfo []goodsinfo
	for queryResults.HasNext() {
		var keym,err = queryResults.Next()
		if err!= nil {
			return shim.Error(err.Error())
		}
		value := keym.Value
		goodsInfo := new(goodsinfo)
		var er = json.Unmarshal(value, goodsInfo)
		if er!=nil {
			return shim.Error(er.Error())
		}
		Goodsinfo = append(Goodsinfo, *goodsInfo)
	}
	var historyInfo = new(historyinfo)
	historyInfo.Goodsinfos = Goodsinfo
	var val, err2 = json.Marshal(historyInfo)
	if err2 != nil {
		return shim.Error(err2.Error())
	} else {
		var v,e =json.Marshal(responseMetadata.Bookmark)
		if e!=nil {
			return shim.Error(e.Error())
		}else {
			val = append(val, v...)
			return shim.Success(val)
		}
	}
}

func main() {
	err:=shim.Start(new(goodsinfo))
	if err !=nil {
		fmt.Println("ebr goodsInfo chaincode start error")
	}
}

