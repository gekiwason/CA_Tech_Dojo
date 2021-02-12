package main
 
import (
    "fmt"
    "log"
    "net/http"
    "crypto/md5"
    "io"

    "github.com/gin-gonic/gin" 
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/mysql"
)

func gormConnect() *gorm.DB {
    DBMS := "mysql"
    USER := "root"
    PASS := "root"
    PROTOCOL := "tcp(localhost:3306)"
    DBNAME := "CA_Tech_Dojo"
    CONNECT := USER + ":" + PASS + "@" + PROTOCOL + "/" + DBNAME
    db, err := gorm.Open(DBMS, CONNECT)

    if err != nil {
        panic(err.Error())
    }
    fmt.Println("db connected: ", &db)
    return db
}

type User struct {
    ID       uint   `gorm:"primaryKey"`
    Name  string `json:name gorm:"unique"`
    Token  string `json:token`
}

func main() {
    log.Println("Server started on: http://localhost:8000")

    r := gin.Default()
    r.POST("/user/create", create)
    r.OPTIONS("/user/create", Options)
    r.Run(":8000")
}

func Options(c *gin.Context) {
	if c.Request.Method != "OPTIONS" {
		c.Next()
	} else {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "authorization, origin, content-type, accept")
		c.Header("Allow", "HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Content-Type", "application/json")
		c.AbortWithStatus(http.StatusOK)
	}
}

func create(c *gin.Context) {
    db := gormConnect()
    db.LogMode(true)
    db.Set("gorm:table_options", "ENGINE=InnoDB")
    db.AutoMigrate(&User{})

    user := User{}
    c.BindJSON(&user)

    token := "q3489fj9q0348nfq034mf"
    token = user.Name + token
    h := md5.New()
    io.WriteString(h, token)
    s := fmt.Sprintf("%x", h.Sum(nil))

    user.Token = s

    db.NewRecord(user)
    db.Create(&user)
    if db.NewRecord(user) == false {
        c.JSON(http.StatusOK, gin.H{
            "token": user.Token,
        })
    }
}
