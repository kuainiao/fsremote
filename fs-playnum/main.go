package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hearts.zhang/fsremote"
)

const fs_playnum = "select mediaid,playnum, daynum,seven_daysnum,weeknum,monthnum,modifydate from fs_media_playnum"

func main() {
	db, err := sql.Open("mysql", "dbs:R4XBfuptAH@tcp(192.168.8.121:3306)/corsair_0")
	panic_error(err)
	defer db.Close()
	rows, err := db.Query(fs_playnum)
	panic_error(err)
	for rows.Next() {
		to := fsremote.FunTomato{}
		//		var id, playnum, daynum, seven_daysnum, weeknum, monthnum int64
		var modifydate []byte
		if err = rows.Scan(&to.MediaId, &to.PlayNum, &to.DayNum, &to.Day7Num, &to.WeekNum, &to.MonthNum, &modifydate); err == nil {
			to.Date = time_parse(string(modifydate)).Unix()
			print_tomato(to)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func print_tomato(to fsremote.FunTomato) {
	if d, err := json.Marshal(to); err == nil {
		fmt.Println(string(d))
	}

}
func panic_error(err error) {
	if err != nil {
		panic(err)
	}
}

//Mon Jan 2 15:04:05 -0700 MST 2006
const tmlayout = "2006-01-02 15:04:05 -0700"

func time_parse(t string) time.Time {
	v, _ := time.Parse(tmlayout, t+" +0800")
	return v
}
