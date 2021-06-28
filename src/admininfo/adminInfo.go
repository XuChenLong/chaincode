package main

import(
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb  "github.com/hyperledger/fabric/protos/peer"
	"strconv"
)

type admininfo struct {
	Identity 		string `json:"identity"`        //管理员编号
	Name            string `json:"name"`            //名称
	Password 	    string `json:"password"`		//密码
	Privatekey      string `json:"privatekey"`      //管理员私钥
	State			string `json:"state"`			//true启用 false禁用
}

type historyinfo struct {
	Admininfo []admininfo `json:"admininfos"`
}

func (t *admininfo)  Init(stub shim.ChaincodeStubInterface) pb.Response {
	_, args := stub.GetFunctionAndParameters()
	err :=stub.PutState(args[0], []byte(args[1]))
	if err!=nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

func (t *admininfo) Invoke(stub shim.ChaincodeStubInterface) pb.Response  {
	var funcName, args = stub.GetFunctionAndParameters()
	if funcName =="create" {
		return t.createAdmin(stub,args)
	}else if funcName =="save" {
		return t.save(stub,args)   //保存货物信息|修改货物信息 #执行前需判断货物state是否为false
	}else if funcName =="query" {
		return t.query(stub,args) //查询货物信息
	}else if funcName == "delete" {
		return t.delete(stub,args)     //删除货物信息 # 执行前需要在sdk中判断该货物state是否为空
	}else if funcName == "queryhistory" {
		return t.queryhistory(stub,args) //查询所有历史记录
	}else if funcName == "querymark" {
		return t.getQueryResult(stub,args) //实现分页查询
	}else if funcName == "findall" {
		return t.findall(stub,args)
	}else{
		return shim.Error("no match function")
	}
}

func (t *admininfo) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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

func (t *admininfo) findall(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var queryString = "{\"selector\":{},\"skip\":"+args[0]+",\"limit\":"+args[1]+"}"
	queryResults,err := stub.GetQueryResult(queryString) //必须是CouchDB才行
	if err!=nil{
		return shim.Error("query failed")
	}
	var Admininfo []admininfo
	for queryResults.HasNext() {
		var keym,err = queryResults.Next()
		if err!= nil {
			return shim.Error(err.Error())
		}
		value := keym.Value
		adminInfo := new(admininfo)
		var er = json.Unmarshal(value, adminInfo)
		if er!=nil {
			return shim.Error(er.Error())
		}
		Admininfo = append(Admininfo, *adminInfo)
	}
	var historyInfo = new(historyinfo)
	historyInfo.Admininfo = Admininfo
	var val, err2 = json.Marshal(historyInfo)
	if err2 != nil {
		return shim.Error(err2.Error())
	}
	return shim.Success(val)
}

func (t *admininfo) save(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=2 {
		return shim.Error("except two args")
	}else {
		var val, err2 = stub.GetState(args[0])
		if err2 !=nil {
			return shim.Error(err2.Error())
		}else if val == nil || string(val) == "{\"identity\":\"unkown\"}" {
			return shim.Error("key never create")
		}
		var err = stub.PutState(args[0], []byte(args[1]))
		if err!=nil {
			return shim.Error(err.Error())
		}else {
			return shim.Success([]byte("Success"))
		}
	}
}

func (t *admininfo) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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

func (t *admininfo) queryhistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=1 {
		return shim.Error("except one arg")
	}else {
		var admininfos []admininfo
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
				var adminInfo = new(admininfo)
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

func (t *admininfo) createAdmin(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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
func (t *admininfo) getQueryResult(stub shim.ChaincodeStubInterface, args []string) pb.Response{
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
	var Admininfo []admininfo
	for queryResults.HasNext() {
		var keym,err = queryResults.Next()
		if err!= nil {
			return shim.Error(err.Error())
		}
		value := keym.Value
		adminInfo := new(admininfo)
		var er = json.Unmarshal(value, adminInfo)
		if er!=nil {
			return shim.Error(er.Error())
		}
		Admininfo = append(Admininfo, *adminInfo)
	}
	var historyInfo = new(historyinfo)
	historyInfo.Admininfo = Admininfo
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
	var err = shim.Start(new(admininfo))
	if err !=nil {
		fmt.Println("ebr adminInfo chaincode start error")
	}
}

