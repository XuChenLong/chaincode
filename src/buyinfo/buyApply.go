package main

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"strconv"
)

type buyapply struct {
	Identity   string `json:"identity"`   //申请编号
	Name       string `json:"name"`       //名称
	Num        int    `json:"num"`        //数量
	Value      int    `json:"value"`	  //价值
	Kind       string `json:"kind"`		  //资产类型
	State      bool   `json:"state"`      //true已经审批 false未审批
	Pass       bool   `json:"pass"`       //审批是否通过 true通过 false否决
	Suggestion string `json:"suggestion"` //回复意见
	Admin      string `json:"adminkey"`   //审批人key
	Userkey    string `json:"userkey"`    //申请人key
	Goodskey   string `json:"goodskey"`   //租借资产key
	Endtime    string `json:"endtime"`    //租借截止时间
	tendtime   string `json:"tendtime"`   //租借时间
}

type historyinfo struct {
	Goodsinfos []buyapply `json:"buyapply"`
}

func (g *buyapply)  Init(shim.ChaincodeStubInterface) pb.Response{
	return shim.Success(nil)
}

func (g *buyapply) Invoke(stub shim.ChaincodeStubInterface) pb.Response  {
	funcName,args := stub.GetFunctionAndParameters()
	if funcName =="create" {
		return g.createApply(stub,args)
	}else if funcName =="save" {
		return g.goodsave(stub,args) //保存货物信息|修改货物信息 #执行前需判断货物state是否为false
	}else if funcName =="query" {
		return g.goodsquery(stub,args) //查询货物信息
	}else if funcName == "delete" {
		return g.delete(stub,args) //删除货物信息 # 执行前需要在sdk中判断该货物state是否为空
	}else if funcName == "queryhistory" {
		return g.queryhistory(stub,args) //查询所有历史记录
	}else if funcName == "querymark" {
		return g.getQueryResult(stub,args) //实现分页查询
	}else{
		return shim.Error("no match function")
	}
}

func (g *buyapply) goodsquery(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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

func (g *buyapply) goodsave(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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

func (g *buyapply) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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

func (g *buyapply) queryhistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=1 {
		return shim.Error("except one arg")
	}else {
		var goodsApplys []buyapply
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
				goodsApply := new(buyapply)
				er :=json.Unmarshal(value,goodsApply)
				if er != nil {
					return shim.Error(er.Error())
				}
				goodsApplys = append(goodsApplys, *goodsApply)
			}
			historyInfo:= new(historyinfo)
			historyInfo.Goodsinfos = goodsApplys
			value,err :=json.Marshal(historyInfo)
			if err!=nil {
				return shim.Error(err.Error())
			}else {
				return shim.Success(value)
			}
		}
	}
}

func (g *buyapply) createApply(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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


//@args[0] 查询项目名称
//@args[1] 查询对象名称
//@args[2] 查询单次数量
//@args[3] 查询的页面hash值 第一次调用可以直接输入空值
//返回对象json数组+书签hash值 书签用于下次查询
func (g *buyapply) getQueryResult(stub shim.ChaincodeStubInterface, args []string) pb.Response{
	if len(args)!=4 {
		return shim.Error("no except four args findname name num bookmark")
	}
	var bookmark = args[3]
	var findname = args[0]
	var name = args[1]
	var pagesize,err = strconv.ParseInt(args[2],10,32)
	if err != nil {
		return shim.Error(err.Error())
	}
	queryString := fmt.Sprintf("{\"selector\":{\"%s\":\"%s\"}}",findname, name)
	queryResults,responseMetadata, err := stub.GetQueryResultWithPagination(queryString,int32(pagesize),bookmark) //必须是CouchDB才行
	if err!=nil{
		return shim.Error("query failed")
	}
	var goodsApplys []buyapply
	for queryResults.HasNext() {
		var keym,err = queryResults.Next()
		if err!= nil {
			return shim.Error(err.Error())
		}
		value := keym.Value
		goodsApply := new(buyapply)
		var er = json.Unmarshal(value,goodsApply)
		if er!=nil {
			return shim.Error(er.Error())
		}
		goodsApplys = append(goodsApplys, *goodsApply)
	}
	var historyInfo = new(historyinfo)
	historyInfo.Goodsinfos = goodsApplys
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
	err:=shim.Start(new(buyapply))
	if err !=nil {
		fmt.Println("ebr goodsApply chaincode start error")
	}
}

