package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"

	log "github.com/sirupsen/logrus"
	jack "gopkg.in/natefinch/lumberjack.v2"
)

//日志保存器
var logger *jack.Logger

var (
	num      = flag.Int("num", 1, "压测数量")
	funcType = flag.Int("func", 8, "压测函数类型")
	// addr     = flag.String("addr", "http://mobiletest.chinaeew.cn:443", "压测地址")
	addr = flag.String("addr", "http://140.210.198.229:80", "压测地址")
)

type loginInfo struct {
	Identification string `json:"identification"` //手机唯一标识
	Phone          string `json:"phone"`          //手机号码
	Password       string `json:"password"`       //密码
}

type markData struct {
	Identification string `json:"identification"` //手机唯一标识
	PushID         string `json:"pushID"`
	Brand          string `json:"brand"` //手机品牌 1:HUAWEI 2:Xiaomi 3:oppo 4:vivo 5:others
}

func initialLogger() {
	//
	logger = &jack.Logger{
		Filename:   "log.txt",
		MaxSize:    10, // megabytes
		MaxBackups: 1000,
		MaxAge:     365,  //days
		Compress:   true, // disabled by default
		LocalTime:  true,
	}
	//
	log.SetOutput(logger)
}

type eWReceoption struct {
	Identification string        `json:"identification"`
	Phone          string        `json:"phone" validate:"required"`   //手机号码 这个值也可能变
	EventID        int64         `json:"eventId" validate:"required"` //地震id
	Statistics     []*statistics `json:"statistics" validate:"required"`
}
type statistics struct {
	Updates            int32   //地震报数
	LocLongitude       float32 //用户经度
	LocLatitude        float32 //用户维度
	Location           string  //用户位置(app上显示的定位)
	Intensity          float32 //烈度
	ReceiveAt          int64   `xorm:"receive_at" json:"receiveAt"` //用户收到此报的时间（距1970的毫秒数）
	Countdown          int32   //用户收到此报后的倒计时(小于0的置0)
	ThresholdMagnitude float32 `xorm:"threshold_magnitude" json:"thresholdMagnitude"` //用户收到此报时的震级阈值
	ThresholdIntensity float32 `xorm:"threshold_intensity" json:"thresholdIntensity"` //用户收到此报时的烈度阈值
}

type safety struct {
	Phone          string `json:"phone"`          //手机号码
	EventID        int64  `json:"eventId"`        //地震id
	Floor          int    `json:"floor"`          //楼层
	Alarm          int    `json:"alarm"`          //是否听到预警警报 0:没有 1:听到了
	SafetyMeasures int    `json:"safetyMeasures"` //是否才去避险措施 0:没有 1:有
	Seisesthesia   int    `json:"seisesthesia"`   //震感 0:没感觉 1:轻微 2:明显 3:剧烈
	Note           string `json:"note"`           //备注
}

var backChan chan int

func main() {
	flag.Parse()
	initialLogger()
	backChan = make(chan int, *num)
	log.Println("start benchmark")
	go func() {
		for i := 0; i < *num; i++ {
			go httptest(i)
		}
	}()
	// continue request
	go func() {
		for v := range backChan {
			go httptest(v)
		}
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c
}

func httptest(i int) {
	defer func() {
		// log.WithField("dandelion", "test").Println("http请求结束", i, time.Now())
		// backChan <- i
	}()
	// log.WithField("dandelion", "test").Println("http请求开始", i, time.Now())
	// num := rand.Intn(70)
	// time.Sleep(time.Duration(num+30) * time.Millisecond)
	client := &http.Client{}
	var data string
	urlAdrr := *addr
	httpFunc := "GET"
	switch *funcType {
	case 0: //拉取地震摘要 分页
		urlAdrr += "/v1/earlywarnings"
		v := url.Values{}
		v.Add("page_size", string(10))
		v.Add("page_num", string(1))
		data = v.Encode()
		break
	case 1: //拉取地震摘要 模拟开始
		urlAdrr += "/v1/earlywarnings"
		v := url.Values{}
		v.Add("start_at", string(0))
		v.Add("updates", string(1))
		data = v.Encode()
		break

	case 2: //拉取地震摘要 不拉取
		urlAdrr += "/v1/earlywarnings"
		v := url.Values{}
		v.Add("start_at", string(1564053255676))
		v.Add("updates", string(1))
		data = v.Encode()
		break
	case 3: //拉取地震摘要 拉取几个
		urlAdrr += "/v1/earlywarnings"
		v := url.Values{}
		v.Add("start_at", string(1561925853000))
		v.Add("updates", string(1))
		data = v.Encode()
		break

	case 4: //拉取地震 1561925853000 3报
		urlAdrr += "/v1/earlywarnings/1561925853000"
		break
	case 5: //拉取地震 1561213795000 17报
		urlAdrr += "/v1/earlywarnings/1561213795000"
		break
	case 6: //拉取地震 1561237704000 7报
		urlAdrr += "/v1/earlywarnings/1561237704000"
		break
	case 7: //拉取版本
		urlAdrr += "/v1/version"
		break
	case 8: //拉取署名
		urlAdrr += "/v1/signature"
		break
	case 9: //拉取公告 分页
		urlAdrr += "/v1/announcements"
		v := url.Values{}
		v.Add("page_size", string(10))
		v.Add("page_num", string(1))
		data = v.Encode()
	case 10: //拉取公告 新版 无数据
		urlAdrr += "/v1/announcements"
		v := url.Values{}
		v.Add("id", string(17))
		data = v.Encode()
	case 11: //拉取公告 新版 有数据
		urlAdrr += "/v1/announcements"
		v := url.Values{}
		v.Add("id", string(15))
		data = v.Encode()
	case 12: //来取科普
		urlAdrr += "/v1/popularization"
		break
	case 13: //来取典型地震 有返回
		urlAdrr += "/v1/model"
		break
	case 14: //登录
		httpFunc = "POST"
		urlAdrr += "/v1/login"
		login := &loginInfo{
			Phone:    "15008417862",
			Password: "123456",
		}
		loginByte, err := json.Marshal(login)
		if err != nil {
			log.WithField("dandelion", "test").Errorln("登录信息转json失败", err.Error())
			return
		}
		data = string(loginByte)
		break
	case 15: //注册
		httpFunc = "POST"
		urlAdrr += "/v1/users"
		login := &loginInfo{
			Identification: "123456",
			Phone:          "15008417862",
			Password:       "123456",
		}
		loginByte, err := json.Marshal(login)
		if err != nil {
			log.WithField("dandelion", "test").Errorln("注册信息转json失败", err.Error())
			return
		}
		data = string(loginByte)
		break
	case 16: //上传push id
		httpFunc = "POST"
		urlAdrr += "/v1/mark"
		mark := &markData{
			Identification: "12345",
			PushID:         "123456",
			Brand:          "HUAWEI",
		}
		loginByte, err := json.Marshal(mark)
		if err != nil {
			log.WithField("dandelion", "test").Errorln("push id信息转json失败", err.Error())
			return
		}
		data = string(loginByte)
		break
	case 17: //上报预警接收回传
		httpFunc = "POST"
		urlAdrr += "/v1/feedback/earlywarnings"
		reception := &eWReceoption{
			Phone:   "15008417862",
			EventID: 1234567,
		}
		stas := &statistics{
			Updates:            1,
			LocLongitude:       103,
			LocLatitude:        31,
			Location:           "四川成都",
			Intensity:          2,
			ReceiveAt:          123456789,
			Countdown:          10,
			ThresholdMagnitude: 4.0,
			ThresholdIntensity: 2.0,
		}
		reception.Statistics = append(reception.Statistics, stas)
		stas1 := *stas
		stas1.Updates = 2
		reception.Statistics = append(reception.Statistics, &stas1)
		stas2 := *stas
		stas1.Updates = 3
		reception.Statistics = append(reception.Statistics, &stas2)
		stas3 := *stas
		stas1.Updates = 4
		reception.Statistics = append(reception.Statistics, &stas3)
		loginByte, err := json.Marshal(reception)
		if err != nil {
			log.WithField("dandelion", "test").Errorln("预警接收反馈信息转json失败", err.Error())
			return
		}
		data = string(loginByte)
		break
	case 18: //上传预警反馈
		httpFunc = "POST"
		urlAdrr += "/v1/feedback/safety"
		safe := &safety{
			Phone:          "15008417862",
			EventID:        123456,
			Floor:          10,
			Alarm:          1,
			SafetyMeasures: 1,
			Seisesthesia:   1,
		}
		safeByte, err := json.Marshal(safe)
		if err != nil {
			log.WithField("dandelion", "test").Errorln("push id信息转json失败", err.Error())
			return
		}
		data = string(safeByte)
		break
	default:
		return
	}

	request, err := http.NewRequest(httpFunc, urlAdrr, strings.NewReader(data))
	if err != nil {
		log.WithField("dandelion", "test").Errorln("创建http请求失败", i, err.Error())
		return
	}

	resp, err := client.Do(request)
	if err != nil {
		log.WithField("dandelion", "xiaomi").Errorln("发送http请求失败", i, err.Error())
		return
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithField("dandelion", "xiaomi").Errorln("读取http响应失败", i, err.Error())
		return
	}
	resp.Body.Close()
}
