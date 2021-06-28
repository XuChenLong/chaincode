package main

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/cid"
	pb "github.com/hyperledger/fabric/protos/peer"
	"strconv"
)

type userinfo struct {
	Username	string				`json:"username"`	//账号
	Name		string				`json:"name"`       //名称
	Goods		map[string]goodmsg	`json:"goods"`		//名下资产
	Msp			string				`json:"msp"`		//Org1MSP - admin Org2MSP - user
}

type goodmsg struct {
	Maxnum    int            `json:"maxnum"`    //admin
	Num       int            `json:"num"`       //user-admin
	Owner     string         `json:"owner"`     //user
	Goodsname string         `json:"goodsname"` //user-admin
	User      map[string]int `json:"user"`      //admin
	State	  bool			 `json:"state"`		//user		同意为true	未同意false
}

type historyinfo struct {
	Userinfos []userinfo `json:"userinfos"`
}

func (t *userinfo) Init(shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (t *userinfo) Invoke(stub shim.ChaincodeStubInterface) pb.Response  {
	var funcName, args = stub.GetFunctionAndParameters()
	if funcName == "create" {			//创造用户
		return t.createUser(stub,args)
	}else if funcName == "return"{		//归还
		return t.userreturn(stub,args)
	}else if funcName == "lenddelete"{	//删除未归还租借
		return t.lenddelete(stub,args)
	}else if funcName == "lend"{		//租借
		return t.lend(stub,args)
	}else if funcName == "checklend"{	//审核租借
		return t.checklend(stub,args)
	}else if funcName == "addgoods" {	//添加资产
		return t.addgoods(stub,args)
	}else if funcName == "deletegoods"{	//删除某一资产
		return t.deletegoods(stub,args)
	}else if funcName == "querygoods"{	//查询username下的goodsmessage
		return t.querygoods(stub,args)
	}else if funcName == "query" {
		return t.queryUser(stub,args)	//查询人员信息
	}else if funcName == "delete" {
		return t.delete(stub,args)		//删除人员信息
	}else if funcName == "queryhistory" {
		return t.queryhistory(stub,args) //查询所有历史记录
	}else if funcName == "querymark" {
		return t.getQueryResult(stub,args)  //实现分页查询
	}else{
		return shim.Error("no such function")
	}
}

//@args[0] name
func (t *userinfo) createUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	msp, _ := cid.GetMSPID(stub)
	uname,err := t.getstubname(stub)
	if err!=nil {
		return shim.Error(err.Error())
	}
	var val, er = stub.GetState(uname)
	if er != nil {
		return shim.Error(er.Error())
	}
	if val	== nil || string(val) == "{\"username\":\"unkown\"}" {
		var user userinfo
		user.Username = uname
		user.Msp = msp
		user.Name = args[0]
		user.Goods = make(map[string]goodmsg)
		value, _ := json.Marshal(user)
		er = stub.PutState(uname, value)
		if er != nil {
			return shim.Error(er.Error())
		}else {
			return shim.Success([]byte("register success"))
		}
	}else {
		return shim.Error("userdata already delete")
	}
}

//删除未通过的租借
//@args[0] admin.goodskey.id
func (t *userinfo) lenddelete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var user userinfo
	msp,_ := cid.GetMSPID(stub)
	if msp == "Org1MSP" {
		return shim.Error("Only user can return")
	}
	uname,err := t.getstubname(stub)
	if err!=nil {
		return shim.Error(err.Error())
	}
	var value, _ = stub.GetState(uname)
	if value==nil {
		return shim.Error("用户不存在")
	}
	_ = json.Unmarshal(value, &user)
	if _,ok :=user.Goods[args[0]];!ok {
		return shim.Error("goods isn't exist")
	}
	if user.Goods[args[0]].State {
		return shim.Error("goods haven't returned")
	}else {
		delete(user.Goods, args[0])
		val, _ :=json.Marshal(user)
		err = stub.PutState(user.Username, val)
		return shim.Success([]byte("lend delete success"))
	}
}

//归还
//@args[0] admin.goodskey.id
func (t *userinfo) userreturn(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var user userinfo
	var admin userinfo
	msp,_ := cid.GetMSPID(stub)
	if msp == "Org1MSP" {
		return shim.Error("Only user can return")
	}
	uname,err := t.getstubname(stub)
	if err!=nil {
		return shim.Error(err.Error())
	}
	var value, _ = stub.GetState(uname)
	if value==nil {
		return shim.Error("用户不存在")
	}
	_ = json.Unmarshal(value, &user)
	if !user.Goods[args[0]].State {
		return shim.Error("goods is not access needn't return")
	}
	value,_ =stub.GetState(user.Goods[args[0]].Owner)
	_ = json.Unmarshal(value, &admin)
	var admingoods = admin.Goods[args[0]]
	admingoods.Num += user.Goods[args[0]].Num
	delete(user.Goods, args[0])
	delete(admin.Goods[args[0]].User,uname)
	admin.Goods[args[0]] = admingoods
	val, _ :=json.Marshal(user)
	err = stub.PutState(user.Username, val)
	if err != nil {
		return shim.Error(err.Error())
	}
	val, _ = json.Marshal(admin)
	err = stub.PutState(admin.Username, val)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte("return success"))
}

//租借
//@args[0] admin.username
//@args[1] admin.goodskey.id
//@args[2] admin.goodskey.num
func (t *userinfo) lend(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var user userinfo
	var admin userinfo
	msp,_ := cid.GetMSPID(stub)
	uname,err := t.getstubname(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if msp == "Org1MSP" {
		return shim.Error("No access")
	}
	var value, er = stub.GetState(args[0])
	if er != nil {
		return shim.Error(er.Error())
	}
	if value == nil {
		return shim.Error("管理员不存在")
	}
	_ = json.Unmarshal(value, &admin)
	if _,ok := admin.Goods[args[1]];!ok {
		return shim.Error("No this goodid")
	}
	value, er = stub.GetState(uname)
	if er != nil {
		return shim.Error(er.Error())
	}
	if value==nil {
		return shim.Error("用户不存在")
	}
	_ = json.Unmarshal(value, &user)
	if _,ok := user.Goods[args[1]];ok {
		return shim.Error("already apply lend")
	}
	num, _ := strconv.Atoi(args[2])
	if admin.Goods[args[1]].Num >= num {
		var usergoods goodmsg
		usergoods.Goodsname = admin.Goods[args[1]].Goodsname
		usergoods.Num = num
		usergoods.Owner = admin.Username
		usergoods.State = false
		user.Goods[args[1]] = usergoods
		val, _ :=json.Marshal(user)
		_ = stub.PutState(uname, val)
		return shim.Success([]byte("lend success"))
	}else {
		return shim.Error("库存数量不足")
	}
}

//审核租借
//@args[0] user.username
//@args[1] admin.goodskey.id
//@args[2] T/F 同意/否决
func (t *userinfo) checklend(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var user userinfo
	var admin userinfo
	msp,_ := cid.GetMSPID(stub)
	if msp == "Org2MSP" {
		return shim.Error("No access")
	}
	aname,err := t.getstubname(stub)
	if err!=nil {
		return shim.Error(err.Error())
	}
	var value, er = stub.GetState(aname)
	if user.Goods[args[1]].State == true{
		return shim.Error("user goodmsg already check")
	}
	if er != nil {
		return shim.Error(er.Error())
	}
	if value == nil {
		return shim.Error("管理员不存在")
	}
	_ = json.Unmarshal(value, &admin)
	if _,ok := admin.Goods[args[1]];!ok {
		return shim.Error("No this goodid")
	}
	value, er = stub.GetState(args[0])
	if er != nil {
		return shim.Error(er.Error())
	}
	if value==nil {
		return shim.Error("用户不存在")
	}
	_ = json.Unmarshal(value, &user)
	if _,ok := user.Goods[args[1]];!ok {
		return shim.Error("No this goodid")
	}
	if user.Goods[args[1]].Owner == admin.Username{
		if args[2] == "F" {
			delete(user.Goods,args[1])
			return shim.Success([]byte("delete lend success"))
		}else if admin.Goods[args[1]].Num >= user.Goods[args[1]].Num {
			var admingoods = admin.Goods[args[1]]
			var usergoods = user.Goods[args[1]]
			usergoods.State = true
			admingoods.Num -= user.Goods[args[1]].Num
			admingoods.User[args[0]] = user.Goods[args[1]].Num
			admin.Goods[args[1]] = admingoods
			user.Goods[args[1]] = usergoods
			val, _ :=json.Marshal(user)
			_ = stub.PutState(user.Username, val)
			val, _ = json.Marshal(admin)
			_ = stub.PutState(admin.Username, val)
			return shim.Success([]byte("check success"))
		}else {
			return shim.Error("库存数量不足")
		}
	}else {
		return shim.Error("No access")
	}
}


//@args[0] goods.id
//@args[1] goods.name
//@args[2] goods.num
func (t *userinfo) addgoods(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	msp, _ := cid.GetMSPID(stub)
	if msp == "Org2MSP" {
		return shim.Error("No access")
	}
	aname,err := t.getstubname(stub)
	if err!=nil {
		return shim.Error(err.Error())
	}
	val,_ := stub.GetState(aname)
	var admin userinfo
	_ = json.Unmarshal(val, &admin)
	if _,ok := admin.Goods[args[0]]; ok {
		return shim.Error("create goodsid is exist")
	}
	var goods goodmsg
	goods.Goodsname = args[1]
	goods.Maxnum, _ = strconv.Atoi(args[2])
	goods.Num = goods.Maxnum
	goods.User = make(map[string]int)
	admin.Goods[args[0]] = goods
	val, _ = json.Marshal(admin)
	err = stub.PutState(aname, val)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte("goods add success"))
}
/*
//@args[0] username
//@args[1] goods.id
//@args[2] goods.num
func (t *userinfo) savegoods(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	msp, _ := cid.GetMSPID(stub)
	val,_ := stub.GetState(args[0])
	var admin userinfo
	_ = json.Unmarshal(val, &admin)
	if admin.Msp != msp || msp != "sa"{
		return shim.Error("No access")
	}
	goods := admin.Goods[args[1]]
	num, _ := strconv.Atoi(args[2])
	if goods.Maxnum - goods.Num > num {
		return shim.Error("调整数量大于当前库存")
	}
	goods.Num = num - goods.Maxnum + goods.Num
	goods.Maxnum = num
	admin.Goods[args[1]] = goods
	val, _ = json.Marshal(admin)
	_ = stub.PutState(admin.Username, val)
	return shim.Success([]byte("goods add success"))
}
*/
//@args[0] username
func (t *userinfo) querygoods(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	val,_ := stub.GetState(args[0])
	var admin userinfo
	_ = json.Unmarshal(val, &admin)
	v, er := json.Marshal(admin.Goods)
	if er != nil {
		return shim.Error(er.Error())
	}
	return shim.Success(v)
}

//@args[0] goods.id
func (t *userinfo) deletegoods(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	msp, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if msp == "Org2MSP" {
		return shim.Error("No access")
	}
	aname,err := t.getstubname(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	val,err := stub.GetState(aname)
	if err != nil {
		return shim.Error(err.Error())
	}
	var admin userinfo
	_ = json.Unmarshal(val, &admin)
	if admin.Goods[args[0]].Num != admin.Goods[args[0]].Maxnum {
		return shim.Error("have goods no return")
	}
	delete(admin.Goods,args[0])
	val, _ = json.Marshal(admin)
	_ = stub.PutState(aname, val)
	return shim.Success([]byte("goods delete success"))
}

//@args[0] goods.id
func (t *userinfo) updatagoods(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	msp, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if msp == "Org2MSP" {
		return shim.Error("No access")
	}
	aname,err := t.getstubname(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	val,err := stub.GetState(aname)
	if err != nil {
		return shim.Error(err.Error())
	}
	var admin userinfo
	_ = json.Unmarshal(val, &admin)
	if admin.Goods[args[0]].Num != admin.Goods[args[0]].Maxnum {
		return shim.Error("have goods no return")
	}
	delete(admin.Goods,args[0])
	val, _ = json.Marshal(admin)
	_ = stub.PutState(aname, val)
	return shim.Success([]byte("goods delete success"))
}

/*
func (t *userinfo) saveUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("except two args")
	}else {
		var val, err2 =stub.GetState(args[0])
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
*/
func (t *userinfo) queryUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=1 {
		return shim.Error("except one args")
	}else {
		var value, err = stub.GetState(args[0])
		if err!=nil {
			shim.Error("no data found")
		}
		return shim.Success(value)
	}
}

//@args[0] 查询项目名称:查询对象名称
//@args[1] 查询单次数量
//@args[2] 查询的页面hash值 第一次调用可以直接输入空值
//返回对象json数组+书签hash值 书签用于下次查询
func (t *userinfo) findall(stub shim.ChaincodeStubInterface, args []string) pb.Response{
	var queryString = "{\"selector\":{},\"skip\":"+args[0]+",\"limit\":"+args[1]+"}"
	queryResults,err := stub.GetQueryResult(queryString) //必须是CouchDB才行
	if err!=nil{
		return shim.Error("query failed")
	}
	var Userinfo []userinfo
	for queryResults.HasNext() {
		var keym,err = queryResults.Next()
		if err!= nil {
			return shim.Error(err.Error())
		}
		value := keym.Value
		userInfo := new(userinfo)
		var er = json.Unmarshal(value, userInfo)
		if er!=nil {
			return shim.Error(er.Error())
		}
		Userinfo = append(Userinfo, *userInfo)
	}
	var historyInfo = new(historyinfo)
	historyInfo.Userinfos = Userinfo
	var val, err2 = json.Marshal(historyInfo)
	if err2 != nil {
		return shim.Error(err2.Error())
	}
	return shim.Success(val)
}


//@args[0] username
func (t *userinfo) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=1 {
		return shim.Error("except one arg")
	}
	msp, _ := cid.GetMSPID(stub)
	if msp == "Org2MSP" {
		return shim.Error("No access")
	}
	var value,err = stub.GetState(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	var check = string(value)
	if check == "{\"username\":\"unkown\"}" {
		return shim.Error("data already delete")
	}
	var duser userinfo
	_ = json.Unmarshal(value, &duser)
	aname,err :=t.getstubname(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if duser.Msp == "Org2MSP" {
		if len(duser.Goods) == 0 {
			err = stub.PutState(args[0],[]byte("{\"username\":\"unkown\"}"))
			return shim.Success([]byte("delete user Success"))
		}else {
			return shim.Error("user has goods not returned")
		}
	}else if duser.Msp == "Org1MSP" &&  aname== "admin"{
		for _,v :=range duser.Goods {
			if v.Num != v.Maxnum {
				return shim.Error("have Goods not returned")
			}
		}
		err = stub.PutState(args[0],[]byte("{\"username\":\"unkown\"}"))
		return shim.Success([]byte("delete admin Success"))
	}else {
		return shim.Error("delete fail")
	}
}

func (t *userinfo) queryhistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!=1 {
		return shim.Error("except one arg")
	}else {
		var userinfos []userinfo
		var it, err = stub.GetHistoryForKey(args[0])
		if err!=nil {
			return shim.Error("no data find")
		}else {
			for it.HasNext() {
				keym,err :=it.Next()
				if err!=nil {
					return shim.Error("data get error")
				}
				value :=keym.Value
				userInfo := new(userinfo)
				er :=json.Unmarshal(value,userInfo)
				if er != nil {
					return shim.Error(er.Error())
				}
				userinfos = append(userinfos, *userInfo)
			}
			var historyInfo = new(historyinfo)
			historyInfo.Userinfos = userinfos
			var value, err = json.Marshal(historyInfo)
			if err!=nil {
				return shim.Error(err.Error())
			}else {
				return shim.Success(value)
			}
		}
	}
}

//@args[0] 查询项目名称:查询对象名称
//@args[1] 查询单次数量
//@args[2] 查询的页面hash值 第一次调用可以直接输入空值
//返回对象json数组+书签hash值 书签用于下次查询
func (t *userinfo) getQueryResult(stub shim.ChaincodeStubInterface, args []string) pb.Response{
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
	var Userinfo []userinfo
	for queryResults.HasNext() {
		var keym,err = queryResults.Next()
		if err!= nil {
			return shim.Error(err.Error())
		}
		value := keym.Value
		userInfo := new(userinfo)
		var er = json.Unmarshal(value, userInfo)
		if er!=nil {
			return shim.Error(er.Error())
		}
		Userinfo = append(Userinfo, *userInfo)
	}
	var historyInfo = new(historyinfo)
	historyInfo.Userinfos = Userinfo
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

func (t *userinfo) getstubname(stub shim.ChaincodeStubInterface)  (string,error) {
	creatorByte,_:= stub.GetCreator()
	certStart := bytes.IndexAny(creatorByte, "-----BEGIN")
	if certStart == -1 {
		return "fail",fmt.Errorf("No certificate found")
	}
	certText := creatorByte[certStart:]
	bl, _ := pem.Decode(certText)
	if bl == nil {
		return "fail",fmt.Errorf("Could not decode the PEM structure")
	}
	cert, err := x509.ParseCertificate(bl.Bytes)
	if err != nil {
		return "fail",fmt.Errorf("ParseCertificate failed")
	}
	uname:=cert.Subject.CommonName
	return uname,nil
}

func main()  {
	var err = shim.Start(new(userinfo))
	if err!=nil {
		fmt.Println("ebr userInfo chaincode start error")
	}
}