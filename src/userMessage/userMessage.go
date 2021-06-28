package main

import(
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb  "github.com/hyperledger/fabric/protos/peer"
)

type usermessage struct {
	Adminkey 	string `json:"adminkey"`  //管理员key
	Name     	string `json:"name"`      //名称
	Num			int    `json:"num"`		  //数量
	State		bool   `json:"state"`	  //审批状态    true已审批 false未审批
	Pass		bool   `json:"pass"`	  //审批是否通过 true通过 false否决
	Suggestion  string `json:"suggestion"`//回复意见
	Userkey  	string `json:"userkey"`   //租借人key
}

type historyinfo struct {
	Admininfo []usermessage `json:"usermessage"`
}

func (g *usermessage)  Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (g *usermessage) Invoke(stub shim.ChaincodeStubInterface) pb.Response  {
	var funcName, args = stub.GetFunctionAndParameters()
	if funcName =="save" {
		return g.save(stub,args)		//保存货物信息|修改货物信息 #执行前需判断货物state是否为false
	}else if funcName =="query" {
		return g.query(stub,args)	//查询货物信息
	}else if funcName == "delete" {
		return g.delete(stub,args)		//删除货物信息 # 执行前需要在sdk中判断该货物state是否为空
	}else if funcName == "queryhistory" {
		return g.queryhistory(stub,args) //查询所有历史记录
	}else{
		return shim.Error("no match function")
	}
}

func (g *usermessage) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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

func (g *usermessage) save(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=2 {
		return shim.Error("except two args")
	}else {
		var err = stub.PutState(args[0], []byte(args[1]))
		if err!=nil {
			return shim.Error(err.Error())
		}else {
			return shim.Success(nil)
		}
	}
}

func (g *usermessage) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=1 {
		return shim.Error("except one arg")
	}else {
		err:=stub.DelState(args[0])
		if err!=nil {
			return shim.Error("data delete fail")
		}else {
			return shim.Success(nil)
		}
	}
}

func (g *usermessage) queryhistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=1 {
		return shim.Error("except one arg")
	}else {
		var admininfos []usermessage
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
				var adminInfo = new(usermessage)
				er :=json.Unmarshal(value,adminInfo)
				if er != nil {
					return shim.Error(er.Error())
				}
				admininfos = append(admininfos, *adminInfo)
			}
			historyInfo:= new(historyinfo)
			historyInfo.Admininfo = admininfos
			value,err :=json.Marshal(historyInfo)
			if err!=nil {
				return shim.Error(err.Error())
			}else {
				return shim.Success(value)
			}
		}
	}
}

func main() {
	var err = shim.Start(new(usermessage))
	if err !=nil {
		fmt.Println("ebr goodsinfo chaincode start error")
	}
}

