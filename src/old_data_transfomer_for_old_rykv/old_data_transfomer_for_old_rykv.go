package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/rykv/kyou"

	_ "github.com/mattn/go-sqlite3"
)

/* メモ
   Mi, TimeIsは、データの状態がUpdateされるので、全部まとめてから処理する必要がある
*/

func main() {
	err := DataTransfer(SrcKyouDir, TranserDestinationDir, "yamato")
	if err != nil {
		panic(err)
	}
}

func init() {
	os.Setenv("HOME", os.Getenv("HOMEPATH"))
}

const TimeLayout = kyou.TimeLayout

var (
	SrcKyouDir            = "$HOME/KyouOld"       // 抽出元のKyouDir
	TranserDestinationDir = "$HOME/GkillTransfer" // 変換済みデータ格納先
)

func DataTransfer(srcKyouDir string, transferDestinationDir string, userName string) error {
	SrcKyouDir = os.ExpandEnv(srcKyouDir)
	srcKyouDir = SrcKyouDir
	TranserDestinationDir = os.ExpandEnv(transferDestinationDir)
	transferDestinationDir = TranserDestinationDir

	err := os.MkdirAll(TranserDestinationDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	// OldのDBから全データをテンポラリDBにいれる
	// 主に（TimeIsとMiのため）
	oldKmemos := []*Kmemo{}
	oldURLogs := []*URLog{}
	oldLantanas := []*Lantana{}
	oldIDFKyous := []*IDFKyou{}
	oldTags := []*Tag{}
	oldTexts := []*Text{}
	oldNlogs := []*Nlog{}
	oldMiTasks := []*MiTask{}
	oldMiCheckStates := []*MiCheckStateInfo{}
	oldMiTitles := []*MiTaskTitleInfo{}
	oldMiLimits := []*MiLimitInfo{}
	oldMiStarts := []*MiStartInfo{}
	oldMiEnds := []*MiEndInfo{}
	oldMiBoards := []*MiBoardInfo{}
	oldTimeIsStarts := []*TimeIsStart{}
	oldTimeIsEnds := []*TimeIsEnd{}

	files, err := os.ReadDir(SrcKyouDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		fmt.Println(filepath.Join(SrcKyouDir, file.Name()))
		log.Printf("★%+v 処理を開始します\n", file.Name())

		repType := strings.Split(file.Name(), "_")[0]
		log.Printf("RepType = %+v\n", repType)

		if file.IsDir() {
			log.Printf("IDFRepDirDir\n")
			_, err := os.Stat(filepath.Join(SrcKyouDir, file.Name(), ".kyou/id.db"))
			if err != nil {
				log.Printf(".kyou/id.dbが存在しません\n")
				log.Printf("スキップします\n")
				continue
			}

			idfKyousFromOldRep, err := getIDFKyousFromOldDB(filepath.Join(SrcKyouDir, file.Name()))
			if err != nil {
				panic(err)
			}

			oldIDFKyous = append(oldIDFKyous, idfKyousFromOldRep...)
			log.Printf("データ抽出:%+v件\n", len(idfKyousFromOldRep))
		} else {
			if !strings.HasSuffix(file.Name(), ".db") {
				log.Printf("スキップします\n")
				continue
			}
			switch repType {
			case "Kmemo":
				log.Printf("Kmemo \n")
				kmemosFromOldRep, err := getKmemosFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldKmemos = append(oldKmemos, kmemosFromOldRep...)
				log.Printf("データ抽出:%+v件\n", len(kmemosFromOldRep))

			case "Lantana":
				log.Printf("Lantana \n")
				lantanasFromOldRep, err := getLantanasFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldLantanas = append(oldLantanas, lantanasFromOldRep...)
				log.Printf("データ抽出:%+v件\n", len(lantanasFromOldRep))

			case "Mi":
				log.Printf("Mi \n")
				miTasksFromOldRep, err := getMiTasksFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldMiTasks = append(oldMiTasks, miTasksFromOldRep...)

				miCheckStatesFromOldRep, err := getMiCheckStatesFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldMiCheckStates = append(oldMiCheckStates, miCheckStatesFromOldRep...)

				miTitlesFromOldRep, err := getMiTaskTitlesFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldMiTitles = append(oldMiTitles, miTitlesFromOldRep...)

				miLimitsFromOldRep, err := getMiLimitsFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldMiLimits = append(oldMiLimits, miLimitsFromOldRep...)

				miStartsFromOldRep, err := getMiStartsFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldMiStarts = append(oldMiStarts, miStartsFromOldRep...)

				miEndsFromOldRep, err := getMiEndsFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldMiEnds = append(oldMiEnds, miEndsFromOldRep...)

				miBoardsFromOldRep, err := getMiBoardsFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldMiBoards = append(oldMiBoards, miBoardsFromOldRep...)

				log.Printf(
					"データ抽出:%+v件\n",
					len(miTasksFromOldRep)+
						len(miCheckStatesFromOldRep)+
						len(miTitlesFromOldRep)+
						len(miLimitsFromOldRep)+
						len(miStartsFromOldRep)+
						len(miEndsFromOldRep)+
						len(miBoardsFromOldRep),
				)

			case "Nlog":
				log.Printf("Nlog \n")
				nlogsFromOldRep, err := getNlogsFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldNlogs = append(oldNlogs, nlogsFromOldRep...)

				log.Printf("データ抽出:%+v件\n", len(nlogsFromOldRep))

			case "Tag":
				log.Printf("Tag \n")
				tagsFromOldRep, err := getTagsFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldTags = append(oldTags, tagsFromOldRep...)

				log.Printf("データ抽出:%+v件\n", len(tagsFromOldRep))

			case "Text":
				log.Printf("Text \n")
				textsFromOldRep, err := getTextsFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldTexts = append(oldTexts, textsFromOldRep...)

				log.Printf("データ抽出:%+v件\n", len(textsFromOldRep))

			case "TimeIs":
				log.Printf("TimeIs \n")
				timeIsStartsFromOldRep, err := getTimeisStartsFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldTimeIsStarts = append(oldTimeIsStarts, timeIsStartsFromOldRep...)

				timeIsEndsFromOldRep, err := getTimeisEndsFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldTimeIsEnds = append(oldTimeIsEnds, timeIsEndsFromOldRep...)

				log.Printf("データ抽出:%+v件\n", len(oldTimeIsStarts)+len(oldTimeIsEnds))

			case "URLog":
				log.Printf("URLog \n")
				urlogsFromOldRep, err := getURLogsFromOldDB(filepath.Join(srcKyouDir, file.Name()))
				if err != nil {
					panic(err)
				}
				oldURLogs = append(oldURLogs, urlogsFromOldRep...)

				log.Printf("データ抽出:%+v件\n", len(oldURLogs))
			default:
			}
		}
	}

	// テンポラリDBから新データを生成、取得する
	tempDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	tempDBInMemory, err := newAllDataDB(tempDB, userName)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertKmemosFromOldDB(oldKmemos)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertURLogsFromOldDB(oldURLogs)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertLantanasFromOldDB(oldLantanas)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertIDFKyousFromOldDB(oldIDFKyous)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertTagsFromOldDB(oldTags)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertTextsFromOldDB(oldTexts)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertNlogsFromOldDB(oldNlogs)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertMiTasksFromOldDB(oldMiTasks)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertMiCheckStatesFromOldDB(oldMiCheckStates)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertMiTaskTitlesFromOldDB(oldMiTitles)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertMiLimitsFromOldDB(oldMiLimits)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertMiStartsFromOldDB(oldMiStarts)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertMiEndsFromOldDB(oldMiEnds)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertMiBoardsFromOldDB(oldMiBoards)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertTimeisStartsFromOldDB(oldTimeIsStarts)
	if err != nil {
		panic(err)
	}
	err = tempDBInMemory.insertTimeisEndsFromOldDB(oldTimeIsEnds)
	if err != nil {
		panic(err)
	}

	// 宛先DBファイルに全部いれる
	kmemos, err := tempDBInMemory.getGkillKmemos()
	if err != nil {
		panic(err)
	}
	for _, kmemo := range kmemos {
		kmemoRep, err := reps.NewKmemoRepositorySQLite3Impl(context.Background(), filepath.Join(transferDestinationDir, kmemo.RepName+".db"))
		if err != nil {
			panic(err)
		}
		err = kmemoRep.AddKmemoInfo(context.Background(), kmemo)
		if err != nil {
			panic(err)
		}
		kmemoRep.Close(context.Background())
	}

	urlogs, err := tempDBInMemory.getGkillURLogs()
	if err != nil {
		panic(err)
	}
	for _, urlog := range urlogs {
		urlogRep, err := reps.NewURLogRepositorySQLite3Impl(context.Background(), filepath.Join(transferDestinationDir, urlog.RepName+".db"))
		if err != nil {
			panic(err)
		}
		err = urlogRep.AddURLogInfo(context.Background(), urlog)
		if err != nil {
			panic(err)
		}
		urlogRep.Close(context.Background())
	}

	lantanas, err := tempDBInMemory.getGkillLantanas()
	if err != nil {
		panic(err)
	}
	for _, lantana := range lantanas {
		lantanaRep, err := reps.NewLantanaRepositorySQLite3Impl(context.Background(), filepath.Join(transferDestinationDir, lantana.RepName+".db"))
		if err != nil {
			panic(err)
		}
		err = lantanaRep.AddLantanaInfo(context.Background(), lantana)
		if err != nil {
			panic(err)
		}
		lantanaRep.Close(context.Background())
	}

	dummyRouter := mux.NewRouter()
	falseValue := false
	idfIgnore := gkill_options.IDFIgnore
	repositoriesRefDummy, err := reps.NewGkillRepositories("")
	if err != nil {
		panic(err)
	}

	idfKyous, err := tempDBInMemory.getGkillIDFKyous()
	if err != nil {
		panic(err)
	}
	for _, idfKyou := range idfKyous {
		err := os.MkdirAll(filepath.Join(transferDestinationDir, idfKyou.RepName, ".gkill/"), os.ModePerm)
		if err != nil {
			panic(err)
		}

		idfKyouRep, err := reps.NewIDFDirRep(context.Background(), filepath.Join(transferDestinationDir, idfKyou.RepName), filepath.Join(transferDestinationDir, idfKyou.RepName, ".gkill/gkill_id.db"), dummyRouter, &falseValue, &idfIgnore, repositoriesRefDummy)
		if err != nil {
			panic(err)
		}
		err = idfKyouRep.AddIDFKyouInfo(context.Background(), idfKyou)
		if err != nil {
			panic(err)
		}
		idfKyouRep.Close(context.Background())
	}

	tags, err := tempDBInMemory.getGkillTags()
	if err != nil {
		panic(err)
	}
	for _, tag := range tags {
		tagRep, err := reps.NewTagRepositorySQLite3Impl(context.Background(), filepath.Join(transferDestinationDir, tag.RepName+".db"))
		if err != nil {
			panic(err)
		}
		err = tagRep.AddTagInfo(context.Background(), tag)
		if err != nil {
			panic(err)
		}
		tagRep.Close(context.Background())
	}

	texts, err := tempDBInMemory.getGkillTexts()
	if err != nil {
		panic(err)
	}
	for _, text := range texts {
		textRep, err := reps.NewTextRepositorySQLite3Impl(context.Background(), filepath.Join(transferDestinationDir, text.RepName+".db"))
		if err != nil {
			panic(err)
		}
		err = textRep.AddTextInfo(context.Background(), text)
		if err != nil {
			panic(err)
		}
		textRep.Close(context.Background())
	}

	nlogs, err := tempDBInMemory.getGkillNlogs()
	if err != nil {
		panic(err)
	}
	for _, nlog := range nlogs {
		nlogRep, err := reps.NewNlogRepositorySQLite3Impl(context.Background(), filepath.Join(transferDestinationDir, nlog.RepName+".db"))
		if err != nil {
			panic(err)
		}
		err = nlogRep.AddNlogInfo(context.Background(), nlog)
		if err != nil {
			panic(err)
		}
		nlogRep.Close(context.Background())
	}

	mis, err := tempDBInMemory.getGkillMis()
	if err != nil {
		panic(err)
	}
	for _, mi := range mis {
		miRep, err := reps.NewMiRepositorySQLite3Impl(context.Background(), filepath.Join(transferDestinationDir, mi.RepName+".db"))
		if err != nil {
			panic(err)
		}
		err = miRep.AddMiInfo(context.Background(), mi)
		if err != nil {
			panic(err)
		}
		miRep.Close(context.Background())
	}

	timeiss, err := tempDBInMemory.getGkillTimeiss()
	if err != nil {
		panic(err)
	}
	for _, timeIs := range timeiss {
		timeIsRep, err := reps.NewTimeIsRepositorySQLite3Impl(context.Background(), filepath.Join(transferDestinationDir, timeIs.RepName+".db"))
		if err != nil {
			panic(err)
		}
		err = timeIsRep.AddTimeIsInfo(context.Background(), timeIs)
		if err != nil {
			panic(err)
		}
		timeIsRep.Close(context.Background())
	}

	return nil
}

type Kmemo struct {
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
	ID      string    `json:"id"`
	RepName string    `json:"rep_name"`
}

type URLog struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Favicon     string    `json:"favicon"`
	Image       string    `json:"image"`
	Time        time.Time `json:"time"`
	ID          string    `json:"id"`
	RepName     string    `json:"rep_name"`
}

type Nlog struct {
	ID       string    `json:"id"`
	Time     time.Time `json:"time"`
	Amount   int       `json:"amount"`
	ShopName string    `json:"shop_name"`
	Memo     string    `json:"memo"`
	RepName  string    `json:"rep_name"`
}

type Lantana struct {
	LantanaID string    `json:"lantana_id"`
	Mood      int       `json:"mood"`
	Time      time.Time `json:"time"`
	RepName   string    `json:"rep_name"`
}

type IDFKyou struct {
	Target  string    `json:"target"`
	ID      string    `json:"id"`
	Time    time.Time `json:"time"`
	LastMod time.Time `json:"last_mod"`
	RepName string    `json:"rep_name"`
}

type Tag struct {
	ID      string    `json:"id"`
	Target  string    `json:"target"`
	Tag     string    `json:"tag"`
	Time    time.Time `json:"time"`
	RepName string    `json:"rep_name"`
}

type Text struct {
	ID      string    `json:"id"`
	Target  string    `json:"target"`
	Text    string    `json:"text"`
	Time    time.Time `json:"time"`
	RepName string    `json:"rep_name"`
}

type MiTask struct {
	TaskID      string    `json:"task_id"`
	CreatedTime time.Time `json:"created_time"`
	RepName     string
}

type MiCheckStateInfo struct {
	CheckStateID string    `json:"check_state_id"`
	TaskID       string    `json:"task_id"`
	UpdatedTime  time.Time `json:"updated_time"`
	IsChecked    bool      `json:"is_checked"`
	RepName      string
}

type MiTaskTitleInfo struct {
	TaskTitleID string    `json:"task_title_id"`
	TaskID      string    `json:"task_id"`
	UpdatedTime time.Time `json:"updated_time"`
	Title       string    `json:"title"`
	RepName     string
}

type MiLimitInfo struct {
	LimitID     string     `json:"limit_id"`
	TaskID      string     `json:"task_id"`
	UpdatedTime time.Time  `json:"updated_time"`
	Limit       *time.Time `json:"limit"`
	RepName     string
}

type MiStartInfo struct {
	StartID     string     `json:"start_id"`
	TaskID      string     `json:"task_id"`
	UpdatedTime time.Time  `json:"updated_time"`
	Start       *time.Time `json:"start"`
	RepName     string
}

type MiEndInfo struct {
	EndID       string     `json:"end_id"`
	TaskID      string     `json:"task_id"`
	UpdatedTime time.Time  `json:"updated_time"`
	End         *time.Time `json:"end"`
	RepName     string
}

type MiBoardInfo struct {
	BoardInfoID string    `json:"board_info_id"`
	TaskID      string    `json:"task_id"`
	UpdatedTime time.Time `json:"updated_time"`
	BoardName   string    `json:"board_name"`
	RepName     string
}

type TimeIsStart struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	StartTime time.Time `json:"start_time"`
	RepName   string
}

type TimeIsEnd struct {
	ID      string     `json:"id"`
	StartID string     `json:"start_id"`
	EndTime *time.Time `json:"end_time"`
	RepName string
}

func getNlogsFromOldDB(filename string) ([]*Nlog, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	statement := `SELECT ID, Time, Amount, ShopName, Memo FROM "nlog";`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at Get from database: %w", err)
		return nil, err
	}
	defer rows.Close()

	nlogs := []*Nlog{}
	repName := RepName(filename)
	for rows.Next() {
		nlog := &Nlog{RepName: repName}
		timestr := ""
		err = rows.Scan(&nlog.ID, &timestr, &nlog.Amount, &nlog.ShopName, &nlog.Memo)
		if err != nil {
			err = fmt.Errorf("error at scan rows from database: %w", err)
			return nil, err
		}
		nlog.Time, err = time.Parse(TimeLayout, strings.ReplaceAll(timestr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time at %s %s: %w", timestr, nlog.ID, err)
			return nil, err
		}
		nlogs = append(nlogs, nlog)
	}
	return nlogs, nil
}

func getKmemosFromOldDB(filename string) ([]*Kmemo, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	statement := `
SELECT
  ID, 
  Content, 
  Time
FROM "kmemo";`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all kmemos from %s: %w", filename, err)
		return nil, err
	}
	defer rows.Close()

	repName := RepName(filename)
	kmemos := []*Kmemo{}
	for rows.Next() {
		kmemo := &Kmemo{RepName: repName}
		timestr := ""
		err = rows.Scan(&kmemo.ID, &kmemo.Content, &timestr)
		if err != nil {
			err = fmt.Errorf("error at scan rows from %s: %w", filename, err)
			return nil, err
		}

		kmemo.Time, err = time.Parse(TimeLayout, strings.ReplaceAll(timestr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s %s: %w", timestr, kmemo.ID, filename, err)
			return nil, err
		}
		kmemos = append(kmemos, kmemo)
	}
	return kmemos, nil
}

func getURLogsFromOldDB(filename string) ([]*URLog, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	statement := `
SELECT
  ID, 
  URL, 
  Title, 
  Description, 
  Favicon, 
  Image, 
  Time
FROM "urlog";`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all urlogs from %s: %w", filename, err)
		return nil, err
	}
	defer rows.Close()

	urlogs := []*URLog{}
	repName := RepName(filename)
	for rows.Next() {
		urlog := &URLog{RepName: repName}
		timestr := ""
		err = rows.Scan(&urlog.ID, &urlog.URL, &urlog.Title, &urlog.Description, &urlog.Favicon, &urlog.Image, &timestr)
		if err != nil {
			err = fmt.Errorf("error at scan rows from %s: %w", filename, err)
			return nil, err
		}
		urlog.Time, err = time.Parse(TimeLayout, strings.ReplaceAll(timestr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s %s: %w", timestr, urlog.ID, filename, err)
			return nil, err
		}
		urlogs = append(urlogs, urlog)
	}
	return urlogs, nil
}

func getLantanasFromOldDB(filename string) ([]*Lantana, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	lantanas := []*Lantana{}
	statement := `
SELECT
    LantanaID,
    Time,
    Mood
FROM
    lantana
`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all lantanas: %w", err)
		return nil, err
	}
	defer rows.Close()

	repName := RepName(filename)
	for rows.Next() {
		lantana := &Lantana{RepName: repName}
		createdTimeStr := ""
		err := rows.Scan(&lantana.LantanaID, &createdTimeStr, &lantana.Mood)
		if err != nil {
			return nil, err
		}

		lantana.Time, err = time.Parse(TimeLayout, strings.ReplaceAll(createdTimeStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time: %w", err)
			return nil, err
		}
		lantanas = append(lantanas, lantana)
	}
	return lantanas, nil
}

func getIDFKyousFromOldDB(filename string) ([]*IDFKyou, error) {
	db, err := sql.Open("sqlite3", filepath.Join(filename, ".kyou/id.db"))
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	fileMetadatas := []*IDFKyou{}
	statement := `SELECT Target, ID, Time, LastMod FROM "id";`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all file metadatas from database %s: %w", filename, err)
		return nil, err
	}
	defer rows.Close()

	repName := RepName(filename)
	// tagを取り出して収める
	for rows.Next() {
		fileMetadata := &IDFKyou{RepName: repName}
		timestr := ""
		lastmodStr := ""
		err = rows.Scan(&fileMetadata.Target, &fileMetadata.ID, &timestr, &lastmodStr)
		if err != nil {
			err = fmt.Errorf("error at scan rows from database %s: %w", filename, err)
			return nil, err
		}
		fileMetadata.Time, err = time.Parse(TimeLayout, strings.ReplaceAll(timestr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s %s: %w", timestr, fileMetadata.ID, filename, err)
			return nil, err
		}
		fileMetadata.LastMod, err = time.Parse(TimeLayout, strings.ReplaceAll(lastmodStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse lastmod time '%s' at %s %s: %w", lastmodStr, fileMetadata, filename, err)
			return nil, err
		}
		fileMetadatas = append(fileMetadatas, fileMetadata)
	}
	return fileMetadatas, nil
}

func getTagsFromOldDB(filename string) ([]*Tag, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	tags := []*Tag{}

	statement := `SELECT ID, Target, Tag, Time FROM "tag";`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all tags from %s: %w", filename, err)
		return nil, err
	}
	defer rows.Close()

	repName := RepName(filename)
	for rows.Next() {
		tag := &Tag{RepName: repName}
		timestr := ""
		err = rows.Scan(&tag.ID, &tag.Target, &tag.Tag, &timestr)
		if err != nil {
			err = fmt.Errorf("error at scan rows from %s: %w", filename, err)
			return nil, err
		}
		tag.Time, err = time.Parse(TimeLayout, strings.ReplaceAll(timestr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s %s: %w", timestr, tag.ID, filename, err)
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func getTextsFromOldDB(filename string) ([]*Text, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	texts := []*Text{}

	statement := `SELECT ID, Target, Text, Time FROM "text";`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all texts from %s: %w", filename, err)
		return nil, err
	}
	defer rows.Close()

	repName := RepName(filename)
	for rows.Next() {
		text := &Text{RepName: repName}
		timestr := ""
		err = rows.Scan(&text.ID, &text.Target, &text.Text, &timestr)
		if err != nil {
			err = fmt.Errorf("error at scan rows from %s: %w", filename, err)
			return nil, err
		}
		text.Time, err = time.Parse(TimeLayout, strings.ReplaceAll(timestr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s %s: %w", timestr, text.ID, filename, err)
			return nil, err
		}
		texts = append(texts, text)
	}
	return texts, nil
}

func getTimeisStartsFromOldDB(filename string) ([]*TimeIsStart, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT start.ID, start.Title, start.StartTime
FROM start;`)
	if err != nil {
		return nil, err
	}

	starts := []*TimeIsStart{}
	repName := RepName(filename)
	for rows.Next() {
		s := &TimeIsStart{RepName: repName}

		startTimeStr := ""
		err := rows.Scan(&s.ID, &s.Title, &startTimeStr)
		if err != nil {
			err = fmt.Errorf("error at scan rows at get history: %w", err)
			return nil, err
		}
		s.StartTime, err = time.Parse(TimeLayout, strings.ReplaceAll(startTimeStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse start time %s: %w", startTimeStr, err)
			return nil, err
		}
		starts = append(starts, s)
	}
	return starts, nil
}

func getTimeisEndsFromOldDB(filename string) ([]*TimeIsEnd, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT end.StartID, end.ID, end.EndTime
FROM end;`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ends := []*TimeIsEnd{}
	repName := RepName(filename)
	for rows.Next() {
		e := &TimeIsEnd{RepName: repName}

		endID, endTimeStr := sql.NullString{}, sql.NullString{}
		err := rows.Scan(&e.StartID, &endID, &endTimeStr)
		if err != nil {
			err = fmt.Errorf("error at scan rows at get history: %w", err)
			return nil, err
		}
		if endID.Valid {
			e.ID = endID.String
			if endTimeStr.Valid {
				e.EndTime = &time.Time{}
				*e.EndTime, err = time.Parse(TimeLayout, strings.ReplaceAll(endTimeStr.String, " ", "T"))
				if err != nil {
					err = fmt.Errorf("error at parse end time %s: %w", endTimeStr, err)
					return nil, err
				}
			}
		}
		ends = append(ends, e)
	}
	return ends, nil
}

func getMiTasksFromOldDB(filename string) ([]*MiTask, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	tasks := []*MiTask{}
	statement := `
SELECT
    TaskID,
    CreatedTime
FROM
    Task;
`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all tasks: %w", err)
		return nil, err
	}
	defer rows.Close()

	repName := RepName(filename)
	for rows.Next() {
		task := &MiTask{RepName: repName}
		createdTimeStr := ""
		err := rows.Scan(&task.TaskID, &createdTimeStr)
		if err != nil {
			return nil, err
		}

		task.CreatedTime, err = time.Parse(TimeLayout, strings.ReplaceAll(createdTimeStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time: %w", err)
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func getMiCheckStatesFromOldDB(filename string) ([]*MiCheckStateInfo, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	statement := `
SELECT
    CheckStateInfo.CheckStateID,
    CheckStateInfo.TaskID,
    CheckStateInfo.UpdatedTime AS UpdatedTime,
	CheckStateInfo.IsChecked
FROM
    CheckStateInfo
`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all check state infos: %w", err)
		return nil, err
	}
	defer rows.Close()

	checkStateInfos := []*MiCheckStateInfo{}

	repName := RepName(filename)
	for rows.Next() {
		checkStateInfo := &MiCheckStateInfo{RepName: repName}
		updatedTimeStr := ""
		err := rows.Scan(&checkStateInfo.CheckStateID,
			&checkStateInfo.TaskID,
			&updatedTimeStr,
			&checkStateInfo.IsChecked,
		)
		if err != nil {
			err = fmt.Errorf("error at get check state info: %w", err)
			return nil, err
		}

		checkStateInfo.UpdatedTime, err = time.Parse(TimeLayout, strings.ReplaceAll(updatedTimeStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time: %w", err)
			return nil, err
		}
		// 現行のCheckState初期値が0（バグなのでここで対応）
		if checkStateInfo.IsChecked {
			checkStateInfo.UpdatedTime = checkStateInfo.UpdatedTime.Add(time.Second * 1)
		}
		checkStateInfos = append(checkStateInfos, checkStateInfo)
	}
	return checkStateInfos, nil
}

func getMiTaskTitlesFromOldDB(filename string) ([]*MiTaskTitleInfo, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	statement := `
SELECT
    TaskTitleID,
    TaskID,
    UpdatedTime,
    Title
FROM
    TaskTitleInfo
`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all task titles: %w", err)
		return nil, err
	}
	defer rows.Close()

	taskTitleInfos := []*MiTaskTitleInfo{}

	for rows.Next() {
		taskTitleInfo := &MiTaskTitleInfo{RepName: RepName(filename)}
		updatedTimeStr := ""
		err := rows.Scan(&taskTitleInfo.TaskTitleID,
			&taskTitleInfo.TaskID,
			&updatedTimeStr,
			&taskTitleInfo.Title)
		if err != nil {
			err = fmt.Errorf("error at get task title info: %w", err)
			return nil, err
		}

		taskTitleInfo.UpdatedTime, err = time.Parse(TimeLayout, strings.ReplaceAll(updatedTimeStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse times: %w", err)
			return nil, err
		}
		taskTitleInfos = append(taskTitleInfos, taskTitleInfo)
	}
	return taskTitleInfos, nil
}

func getMiLimitsFromOldDB(filename string) ([]*MiLimitInfo, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	statement := `
SELECT
    LimitID,
    TaskID,
    UpdatedTime,
    LimitTime
FROM
    LimitInfo
`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all limits: %w", err)
		return nil, err
	}
	defer rows.Close()

	limitInfos := []*MiLimitInfo{}
	for rows.Next() {
		limitInfo := &MiLimitInfo{RepName: RepName(filename)}
		updatedTimeStr := ""
		limitTimeStr := sql.NullString{}
		err := rows.Scan(&limitInfo.LimitID,
			&limitInfo.TaskID,
			&updatedTimeStr,
			&limitTimeStr)
		if err != nil {
			err = fmt.Errorf("error at get limit info: %w", err)
			return nil, err
		}

		limitInfo.UpdatedTime, err = time.Parse(TimeLayout, strings.ReplaceAll(updatedTimeStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time: %w", err)
			return nil, err
		}

		if limitTimeStr.Valid {
			limitInfo.Limit = &time.Time{}
			*limitInfo.Limit, err = time.Parse(TimeLayout, limitTimeStr.String)
			if err != nil {
				err = fmt.Errorf("error at parse limit times: %w", err)
				return nil, err
			}
		}
		limitInfos = append(limitInfos, limitInfo)
	}
	return limitInfos, nil
}

func getMiStartsFromOldDB(filename string) ([]*MiStartInfo, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	statement := `
SELECT
    StartID,
    TaskID,
    UpdatedTime,
    StartTime
FROM
    StartInfo
`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all mi starts: %w", err)
		return nil, err
	}
	defer rows.Close()

	startInfos := []*MiStartInfo{}

	for rows.Next() {
		startInfo := &MiStartInfo{RepName: RepName(filename)}
		updatedTimeStr := ""
		startTimeStr := sql.NullString{}
		err := rows.Scan(&startInfo.StartID,
			&startInfo.TaskID,
			&updatedTimeStr,
			&startTimeStr)
		if err != nil {
			err = fmt.Errorf("error at get start info: %w", err)
			return nil, err
		}

		startInfo.UpdatedTime, err = time.Parse(TimeLayout, strings.ReplaceAll(updatedTimeStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time: %w", err)
			return nil, err
		}

		if startTimeStr.Valid {
			startInfo.Start = &time.Time{}
			*startInfo.Start, err = time.Parse(TimeLayout, startTimeStr.String)
			if err != nil {
				err = fmt.Errorf("error at parse start time : %w", err)
				return nil, err
			}
		}
		startInfos = append(startInfos, startInfo)
	}
	return startInfos, nil
}

func getMiEndsFromOldDB(filename string) ([]*MiEndInfo, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	statement := `
SELECT
    EndID,
    TaskID,
    UpdatedTime,
    EndTime
FROM
    EndInfo
`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all mi starts: %w", err)
		return nil, err
	}
	defer rows.Close()

	endInfos := []*MiEndInfo{}

	for rows.Next() {
		miEndInfo := &MiEndInfo{RepName: RepName(filename)}
		updatedTimeStr := ""
		endTimeStr := sql.NullString{}
		err := rows.Scan(&miEndInfo.EndID,
			&miEndInfo.TaskID,
			&updatedTimeStr,
			&endTimeStr)
		if err != nil {
			err = fmt.Errorf("error at get limit info: %w", err)
			return nil, err
		}

		miEndInfo.UpdatedTime, err = time.Parse(TimeLayout, strings.ReplaceAll(updatedTimeStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time: %w", err)
			return nil, err
		}

		if endTimeStr.Valid {
			miEndInfo.End = &time.Time{}
			*miEndInfo.End, err = time.Parse(TimeLayout, endTimeStr.String)
			if err != nil {
				err = fmt.Errorf("error at parse limit time: %w", err)
				return nil, err
			}
		}
		endInfos = append(endInfos, miEndInfo)
	}
	return endInfos, nil
}

func getMiBoardsFromOldDB(filename string) ([]*MiBoardInfo, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}
	defer db.Close()

	statement := `
SELECT
    BoardInfoID,
    TaskID,
    UpdatedTime,
    BoardName
FROM
    BoardInfo
`
	rows, err := db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all mi starts: %w", err)
		return nil, err
	}
	defer rows.Close()

	boardInfos := []*MiBoardInfo{}

	for rows.Next() {
		boardInfo := &MiBoardInfo{RepName: RepName(filename)}
		updatedTimeStr := ""
		err := rows.Scan(&boardInfo.BoardInfoID,
			&boardInfo.TaskID,
			&updatedTimeStr,
			&boardInfo.BoardName)
		if err != nil {
			err = fmt.Errorf("error at get board info: %w", err)
			return nil, err
		}

		boardInfo.UpdatedTime, err = time.Parse(TimeLayout, strings.ReplaceAll(updatedTimeStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse time: %w", err)
			return nil, err
		}
		boardInfos = append(boardInfos, boardInfo)
	}
	return boardInfos, nil
}

type allDataDB struct {
	db       *sql.DB
	UserName string
}

func newAllDataDB(db *sql.DB, userName string) (*allDataDB, error) {
	allDataDB := &allDataDB{
		db:       db,
		UserName: userName,
	}
	var err error
	_, err = db.Exec(`PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS "kmemo" (ID NOT NULL, Content NOT NULL, Time NOT NULL, RepName NOT NULL);`)
	if err != nil {
		err = fmt.Errorf("error at create kmemo db")
		err = fmt.Errorf("error at PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE: %w", err)
		return nil, err
	}
	_, err = db.Exec(`PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS "urlog" (ID NOT NULL, URL NOT NULL, Title NOT NULL, Description NOT NULL, Favicon NOT NULL, Image NOT NULL, Time NOT NULL, RepName NOT NULL);`)
	if err != nil {
		err = fmt.Errorf("error at create urlog db")
		err = fmt.Errorf("error at PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE to database: %w", err)
		return nil, err
	}

	_, err = db.Exec(`PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS "nlog" (ID NOT NULL, Time NOT NULL, Amount NOT NULL, Memo NOT NULL, ShopName NOT NULL, RepName NOT NULL);`)
	if err != nil {
		err = fmt.Errorf("error at create nlog db")
		err = fmt.Errorf("error at PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE to database: %w", err)
		return nil, err
	}

	_, err = db.Exec(`
PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS lantana (LantanaID TEXT NOT NULL, Time TEXT NOT NULL, Mood INTEGER NOT NULL, RepName NOT NULL);
	`)
	if err != nil {
		err = fmt.Errorf("error at create lantana db")
		err = fmt.Errorf("error at PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE to database: %w", err)
		return nil, err
	}
	_, err = db.Exec(`PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS "id" (Target NOT NULL, ID NOT NULL, Time NOT NULL, LastMod NOT NULL, RepName NOT NULL);`)
	if err != nil {
		err = fmt.Errorf("error at create id db")
		err = fmt.Errorf("error at PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE to database: %w", err)
		return nil, err
	}
	_, err = db.Exec(`PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS "tag" (ID NOT NULL, Target NOT NULL, Tag NOT NULL, Time NOT NULL, RepName NOT NULL);`)
	if err != nil {
		err = fmt.Errorf("error at create tag db")
		err = fmt.Errorf("error at PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE to database: %w", err)
		return nil, err
	}
	_, err = db.Exec(`PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS "text" (ID NOT NULL, Text NOT NULL, Target NOT NULL, Time NOT NULL, RepName NOT NULL);`)
	if err != nil {
		err = fmt.Errorf("error at create text db")
		err = fmt.Errorf("error at PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE to database: %w", err)
		return nil, err
	}
	_, err = db.Exec(`PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS "timeis_start" (
		ID        TEXT NOT NULL,
		Title     TEXT NOT NULL,
		StartTime TEXT NOT NULL,
		RepName TEXT NOT NULL
	);`)
	if err != nil {
		err = fmt.Errorf("error at timeis start db")
		err = fmt.Errorf("error at create start table: %w", err)
		return nil, err
	}
	_, err = db.Exec(`PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS "timeis_end" (
		ID      TEXT NOT NULL,
		StartID NOT NULL,
		EndTime TEXT,
		RepName TEXT NOT NULL
	);`)
	if err != nil {
		err = fmt.Errorf("error at timeis end db")
		err = fmt.Errorf("error at create start table: %w", err)
		return nil, err
	}
	_, err = db.Exec(`
PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS Task (
    TaskID TEXT NOT NULL,
    CreatedTime TEXT NOT NULL,
	RepName TEXT NOT NULL
);`)
	if err != nil {
		err = fmt.Errorf("error at timeis end db")
		err = fmt.Errorf("error at create start table: %w", err)
		return nil, err
	}
	_, err = db.Exec(`

PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS TaskTitleInfo (
    TaskTitleID TEXT NOT NULL,
    TaskID TEXT NOT NULL,
    UpdatedTime TEXT NOT NULL,
    Title TEXT NOT NULL,
	RepName TEXT NOT NULL
);`)
	if err != nil {
		err = fmt.Errorf("error at timeis end db")
		err = fmt.Errorf("error at create start table: %w", err)
		return nil, err
	}
	_, err = db.Exec(`

PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS CheckStateInfo (
    CheckStateID TEXT NOT NULL,
    TaskID TEXT NOT NULL,
    UpdatedTime TEXT NOT NULL,
    IsChecked TEXT NOT NULL,
	RepName TEXT NOT NULL
);`)
	if err != nil {
		err = fmt.Errorf("error at timeis end db")
		err = fmt.Errorf("error at create start table: %w", err)
		return nil, err
	}
	_, err = db.Exec(`

PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS LimitInfo (
    LimitID TEXT NOT NULL,
    TaskID TEXT NOT NULL,
    UpdatedTime TEXT NOT NULL,
    LimitTime Text,
	RepName TEXT NOT NULL
);`)
	if err != nil {
		err = fmt.Errorf("error at timeis end db")
		err = fmt.Errorf("error at create start table: %w", err)
		return nil, err
	}
	_, err = db.Exec(`

PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS MiStartInfo (
    StartID TEXT NOT NULL,
    TaskID TEXT NOT NULL,
    UpdatedTime TEXT NOT NULL,
    StartTime Text,
	RepName TEXT NOT NULL
);`)
	if err != nil {
		err = fmt.Errorf("error at timeis end db")
		err = fmt.Errorf("error at create start table: %w", err)
		return nil, err
	}
	_, err = db.Exec(`

PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS MiEndInfo (
    EndID TEXT NOT NULL,
    TaskID TEXT NOT NULL,
    UpdatedTime TEXT NOT NULL,
    EndTime Text,
	RepName TEXT NOT NULL
);`)
	if err != nil {
		err = fmt.Errorf("error at timeis end db")
		err = fmt.Errorf("error at create start table: %w", err)
		return nil, err
	}
	_, err = db.Exec(`

PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE IF NOT EXISTS BoardInfo (
    BoardInfoID TEXT NOT NULL,
    TaskID TEXT NOT NULL,
    UpdatedTime TEXT NOT NULL,
	RepName TEXT NOT NULL,
    BoardName TEXT NOT NULL
);
`)
	if err != nil {
		err = fmt.Errorf("error at mi db")
		err = fmt.Errorf("error at PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -50000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;VACUUM;
CREATE TABLE to database: %w", err)
		return nil, err
	}
	return allDataDB, nil
}

func (a *allDataDB) insertNlogsFromOldDB(nlogs []*Nlog) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	statement := `INSERT INTO "nlog" (
	ID, 
	Time, 
	Amount, 
	Memo, 
	ShopName, 
	RepName
) VALUES(
	?,
	?,
	?,
	?,
	?,
	?)`
	for _, nlog := range nlogs {
		_, err := tx.Exec(statement, []interface{}{
			nlog.ID,
			nlog.Time.Format(TimeLayout),
			nlog.Amount,
			nlog.Memo,
			nlog.ShopName,
			nlog.RepName,
		}...)
		if err != nil {
			err = fmt.Errorf("error at add nlog to to database: %w", err)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertKmemosFromOldDB(kmemos []*Kmemo) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO kmemo(
			ID,
			Content,
			Time,
			RepName
		  ) VALUES (
			  ?,
			  ?,
			  ?,
			  ?
		  )`,
	)
	for _, kmemo := range kmemos {
		_, err = tx.Exec(sql, []interface {
		}{
			kmemo.ID,
			kmemo.Content,
			kmemo.Time.Format(TimeLayout),
			kmemo.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertURLogsFromOldDB(urlogs []*URLog) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO urlog (
			ID,
			URL,
			Title,
			Description,
			Favicon,
			Image,
			Time,
			RepName
		  ) VALUES (
		    ?,
		    ?,
		    ?,
		    ?,
		    ?,
		    ?,
		    ?,
		    ?
		  )`,
	)
	for _, urlog := range urlogs {
		_, err = tx.Exec(sql, []interface {
		}{
			urlog.ID,
			urlog.URL,
			urlog.Title,
			urlog.Description,
			urlog.Favicon,
			urlog.Image,
			urlog.Time.Format(TimeLayout),
			urlog.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertLantanasFromOldDB(lantanas []*Lantana) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO lantana (
		  LantanaID,
		  Time,
		  Mood,
		  RepName
		) VALUES (
		  ?,
		  ?,
		  ?,
		  ?
		)`,
	)
	for _, lantana := range lantanas {
		_, err = tx.Exec(sql, []interface {
		}{
			lantana.LantanaID,
			lantana.Time.Format(TimeLayout),
			lantana.Mood,
			lantana.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertIDFKyousFromOldDB(idfKyous []*IDFKyou) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO id (
			Target,
			ID,
			Time,
			LastMod,
			RepName
		) VALUES (
			?,
			?,
			?,
			?,
			?
		)`,
	)
	for _, idfKyou := range idfKyous {
		_, err = tx.Exec(sql, []interface {
		}{
			idfKyou.Target,
			idfKyou.ID,
			idfKyou.Time.Format(TimeLayout),
			idfKyou.LastMod.Format(TimeLayout),
			idfKyou.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertTagsFromOldDB(tags []*Tag) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO tag (
			ID,
			Target,
			Tag,
			Time,
			RepName
		) VALUES (
			?,
			?,
			?,
			?,
			?
		)`,
	)
	for _, tag := range tags {
		_, err = tx.Exec(sql, []interface {
		}{
			tag.ID,
			tag.Target,
			tag.Tag,
			tag.Time.Format(TimeLayout),
			tag.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertTextsFromOldDB(texts []*Text) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO "text" (
			ID,
			Target,
			Text,
			Time,
			RepName
		) VALUES (
			?,
			?,
			?,
			?,
			?
		)`,
	)
	for _, text := range texts {
		_, err = tx.Exec(sql, []interface {
		}{
			text.ID,
			text.Target,
			text.Text,
			text.Time.Format(TimeLayout),
			text.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertTimeisStartsFromOldDB(timeisStarts []*TimeIsStart) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO timeis_start (
			ID,
			Title,
			StartTime,
			RepName
		) VALUES (
			?,
			?,
			?,
			?
		)`,
	)
	for _, timeisStart := range timeisStarts {
		_, err = tx.Exec(sql, []interface {
		}{
			timeisStart.ID,
			timeisStart.Title,
			timeisStart.StartTime.Format(TimeLayout),
			timeisStart.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertTimeisEndsFromOldDB(timeisEnds []*TimeIsEnd) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql_ := fmt.Sprintf(
		`INSERT INTO timeis_end (
			ID,
			StartID,
			EndTime,
			RepName
		) VALUES (
			?,
			?,
			?,
			?
		)`,
	)
	for _, timeisEnd := range timeisEnds {
		end := sql.NullString{}
		if timeisEnd.EndTime != nil {
			end.String = timeisEnd.EndTime.Format(TimeLayout)
			end.Valid = true
		}

		_, err = tx.Exec(sql_, []interface {
		}{
			timeisEnd.ID,
			timeisEnd.StartID,
			end,
			timeisEnd.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertMiTasksFromOldDB(miTasks []*MiTask) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO Task (
			TaskID,
			CreatedTime,
			RepName
		) VALUES (
			?,
			?,
			?
		)`,
	)
	for _, miTask := range miTasks {
		_, err = tx.Exec(sql, []interface {
		}{
			miTask.TaskID,
			miTask.CreatedTime.Format(TimeLayout),
			miTask.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertMiCheckStatesFromOldDB(miCheckStateInfos []*MiCheckStateInfo) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO CheckStateInfo (
			CheckStateID,
			TaskID,
			UpdatedTime,
			IsChecked,
			RepName
		) VALUES (
			?,
			?,
			?,
			?,
			?
		)`,
	)
	for _, miCheckStateInfo := range miCheckStateInfos {
		_, err = tx.Exec(sql, []interface {
		}{
			miCheckStateInfo.CheckStateID,
			miCheckStateInfo.TaskID,
			miCheckStateInfo.UpdatedTime.Format(TimeLayout),
			miCheckStateInfo.IsChecked,
			miCheckStateInfo.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertMiTaskTitlesFromOldDB(miTaskTitleInfos []*MiTaskTitleInfo) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO TaskTitleInfo (
			TaskTitleID,
			TaskID,
			UpdatedTime,
			Title,
			RepName
		) VALUES (
			?,
			?,
			?,
			?,
			?
		)`,
	)
	for _, miTaskTitleInfo := range miTaskTitleInfos {
		_, err = tx.Exec(sql, []interface {
		}{
			miTaskTitleInfo.TaskTitleID,
			miTaskTitleInfo.TaskID,
			miTaskTitleInfo.UpdatedTime.Format(TimeLayout),
			miTaskTitleInfo.Title,
			miTaskTitleInfo.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertMiLimitsFromOldDB(miLimitInfos []*MiLimitInfo) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql_ := fmt.Sprintf(
		`INSERT INTO LimitInfo (
			LimitID,
			TaskID,
			UpdatedTime,
			LimitTime,
			RepName
		) VALUES (
			?,
			?,
			?,
			?,
			?
		)`,
	)
	for _, miLimitInfo := range miLimitInfos {
		limit := sql.NullString{}
		if miLimitInfo.Limit != nil {
			limit.String = miLimitInfo.Limit.Format(TimeLayout)
			limit.Valid = true
		}

		_, err = tx.Exec(sql_, []interface {
		}{
			miLimitInfo.LimitID,
			miLimitInfo.TaskID,
			miLimitInfo.UpdatedTime.Format(TimeLayout),
			limit,
			miLimitInfo.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertMiStartsFromOldDB(miStartInfos []*MiStartInfo) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql_ := fmt.Sprintf(
		`INSERT INTO MiStartInfo (
			StartID,
			TaskID,
			UpdatedTime,
			StartTime,
			RepName
		) VALUES (
			?,
			?,
			?,
			?,
			?
		)`,
	)
	for _, miStartInfo := range miStartInfos {
		start := sql.NullString{}
		if miStartInfo.Start != nil {
			start.String = miStartInfo.Start.Format(TimeLayout)
			start.Valid = true
		}

		_, err = tx.Exec(sql_, []interface {
		}{
			miStartInfo.StartID,
			miStartInfo.TaskID,
			miStartInfo.UpdatedTime.Format(TimeLayout),
			start,
			miStartInfo.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertMiEndsFromOldDB(miEndInfos []*MiEndInfo) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql_ := fmt.Sprintf(
		`INSERT INTO MiEndInfo (
			EndID,
			TaskID,
			UpdatedTime,
			EndTime,
			RepName
		) VALUES (
			?,
			?,
			?,
			?,
			?
		)`,
	)
	for _, miEndInfo := range miEndInfos {
		end := sql.NullString{}
		if miEndInfo.End != nil {
			end.String = miEndInfo.End.Format(TimeLayout)
			end.Valid = true
		}

		_, err = tx.Exec(sql_, []interface {
		}{
			miEndInfo.EndID,
			miEndInfo.TaskID,
			miEndInfo.UpdatedTime.Format(TimeLayout),
			end,
			miEndInfo.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) insertMiBoardsFromOldDB(miBoardInfos []*MiBoardInfo) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`INSERT INTO BoardInfo (
			BoardInfoID,
			TaskID,
			UpdatedTime,
			BoardName,
			RepName
		) VALUES (
			?,
			?,
			?,
			?,
			?
		)`,
	)
	for _, miBoardInfo := range miBoardInfos {
		_, err = tx.Exec(sql, []interface {
		}{
			miBoardInfo.BoardInfoID,
			miBoardInfo.TaskID,
			miBoardInfo.UpdatedTime.Format(TimeLayout),
			miBoardInfo.BoardName,
			miBoardInfo.RepName,
		}...)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (a *allDataDB) getGkillNlogs() ([]*reps.Nlog, error) {
	nlogs := []*reps.Nlog{}
	statement := `
SELECT
  ID, 
  Time,
  Amount,
  ShopName,
  Memo,
  RepName
FROM "nlog";`
	rows, err := a.db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all nlogs: %w", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		nlog := &reps.Nlog{}
		timestr := ""
		err = rows.Scan(&nlog.ID, &timestr, &nlog.Amount, &nlog.Shop, &nlog.Title, &nlog.RepName)
		if err != nil {
			err = fmt.Errorf("error at scan rows: %w", err)
			return nil, err
		}

		time, err := time.Parse(TimeLayout, timestr)
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s: %w", timestr, nlog.ID, err)
			return nil, err
		}
		nlog.IsDeleted = false
		nlog.CreateApp = "nlog"
		nlog.CreateDevice = strings.Split(nlog.RepName, "_")[1]
		nlog.CreateUser = a.UserName
		nlog.UpdateApp = "nlog"
		nlog.UpdateDevice = strings.Split(nlog.RepName, "_")[1]
		nlog.UpdateUser = a.UserName
		nlog.CreateTime = time
		nlog.UpdateTime = time
		nlog.RelatedTime = time
		nlogs = append(nlogs, nlog)
	}
	return nlogs, nil
}

func (a *allDataDB) getGkillKmemos() ([]*reps.Kmemo, error) {
	kmemos := []*reps.Kmemo{}
	statement := `
SELECT
  ID, 
  Content, 
  Time,
  RepName
FROM "kmemo";`
	rows, err := a.db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all kmemos: %w", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		kmemo := &reps.Kmemo{}
		timestr := ""
		err = rows.Scan(&kmemo.ID, &kmemo.Content, &timestr, &kmemo.RepName)
		if err != nil {
			err = fmt.Errorf("error at scan rows: %w", err)
			return nil, err
		}

		time, err := time.Parse(TimeLayout, timestr)
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s: %w", timestr, kmemo.ID, err)
			return nil, err
		}
		kmemo.IsDeleted = false
		kmemo.CreateApp = "kmemo"
		kmemo.CreateDevice = strings.Split(kmemo.RepName, "_")[1]
		kmemo.CreateUser = a.UserName
		kmemo.UpdateApp = "kmemo"
		kmemo.UpdateDevice = strings.Split(kmemo.RepName, "_")[1]
		kmemo.UpdateUser = a.UserName
		kmemo.CreateTime = time
		kmemo.UpdateTime = time
		kmemo.RelatedTime = time
		kmemos = append(kmemos, kmemo)
	}
	return kmemos, nil
}

func (a *allDataDB) getGkillURLogs() ([]*reps.URLog, error) {
	statement := `
SELECT
  ID, 
  URL, 
  Title, 
  Description, 
  Favicon, 
  Image, 
  Time,
  RepName
FROM "urlog";`
	rows, err := a.db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all urlogs: %w", err)
		return nil, err
	}
	defer rows.Close()

	urlogs := []*reps.URLog{}
	for rows.Next() {
		urlog := &reps.URLog{}
		timestr := ""
		err = rows.Scan(&urlog.ID, &urlog.URL, &urlog.Title, &urlog.Description, &urlog.FaviconImage, &urlog.ThumbnailImage, &timestr, &urlog.RepName)
		if err != nil {
			err = fmt.Errorf("error at scan rows: %w", err)
			return nil, err
		}
		time, err := time.Parse(TimeLayout, timestr)
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s: %w", timestr, urlog.ID, err)
			return nil, err
		}
		urlog.IsDeleted = false
		urlog.CreateApp = "urlog"
		urlog.CreateDevice = strings.Split(urlog.RepName, "_")[1]
		urlog.CreateUser = a.UserName
		urlog.UpdateApp = "urlog"
		urlog.UpdateDevice = strings.Split(urlog.RepName, "_")[1]
		urlog.UpdateUser = a.UserName
		urlog.CreateTime = time
		urlog.UpdateTime = time
		urlog.RelatedTime = time
		urlogs = append(urlogs, urlog)
	}
	return urlogs, nil
}

func (a *allDataDB) getGkillLantanas() ([]*reps.Lantana, error) {
	lantanas := []*reps.Lantana{}
	statement := `
SELECT
    LantanaID,
    Time,
    Mood,
	RepName
FROM
    lantana
`
	rows, err := a.db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all lantanas: %w", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		lantana := &reps.Lantana{}
		createdTimeStr := ""
		err := rows.Scan(&lantana.ID, &createdTimeStr, &lantana.Mood, &lantana.RepName)
		if err != nil {
			return nil, err
		}

		time, err := time.Parse(TimeLayout, createdTimeStr)
		if err != nil {
			err = fmt.Errorf("error at parse time: %w", err)
			return nil, err
		}
		lantana.IsDeleted = false
		lantana.CreateApp = "lantana"
		lantana.CreateDevice = strings.Split(lantana.RepName, "_")[1]
		lantana.CreateUser = a.UserName
		lantana.UpdateApp = "lantana"
		lantana.UpdateDevice = strings.Split(lantana.RepName, "_")[1]
		lantana.UpdateUser = a.UserName
		lantana.CreateTime = time
		lantana.UpdateTime = time
		lantana.RelatedTime = time
		lantanas = append(lantanas, lantana)
	}
	return lantanas, nil
}

func (a *allDataDB) getGkillIDFKyous() ([]*reps.IDFKyou, error) {
	fileMetadatas := []*reps.IDFKyou{}

	statement := `SELECT Target, ID, Time, LastMod, RepName FROM "id";`
	rows, err := a.db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all file metadatas from database: %w", err)
		return nil, err
	}
	defer rows.Close()

	// tagを取り出して収める
	for rows.Next() {
		fileMetadata := &reps.IDFKyou{}
		timestr := ""
		lastmodStr := ""
		err = rows.Scan(&fileMetadata.TargetFile, &fileMetadata.ID, &timestr, &lastmodStr, &fileMetadata.RepName)
		if err != nil {
			err = fmt.Errorf("error at scan rows from database: %w", err)
			return nil, err
		}
		fileMetadata.RelatedTime, err = time.Parse(TimeLayout, lastmodStr)
		if err != nil {
			err = fmt.Errorf("error at parse lastmod time '%s' at %s: %w", lastmodStr, fileMetadata, err)
			return nil, err
		}
		time, err := time.Parse(TimeLayout, timestr)
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s: %w", timestr, fileMetadata.ID, err)
			return nil, err
		}
		fileMetadata.IsDeleted = false
		fileMetadata.CreateApp = "idf"
		fileMetadata.CreateDevice = strings.Split(fileMetadata.RepName, "_")[1]
		fileMetadata.CreateUser = a.UserName
		fileMetadata.UpdateApp = "idf"
		fileMetadata.UpdateDevice = strings.Split(fileMetadata.RepName, "_")[1]
		fileMetadata.UpdateUser = a.UserName
		fileMetadata.CreateTime = time
		fileMetadata.UpdateTime = time
		fileMetadata.RelatedTime = time
		fileMetadatas = append(fileMetadatas, fileMetadata)
	}
	return fileMetadatas, nil
}

func (a *allDataDB) getGkillTags() ([]*reps.Tag, error) {
	tags := []*reps.Tag{}

	statement := `SELECT ID, Target, Tag, Time, RepName FROM "tag";`
	rows, err := a.db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all tags: %w", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		tag := &reps.Tag{}
		timestr := ""
		err = rows.Scan(&tag.ID, &tag.TargetID, &tag.Tag, &timestr, &tag.RepName)
		if err != nil {
			err = fmt.Errorf("error at scan rows: %w", err)
			return nil, err
		}
		time, err := time.Parse(TimeLayout, timestr)
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s: %w", timestr, tag.ID, err)
			return nil, err
		}
		tag.IsDeleted = false
		tag.CreateApp = "tag"
		tag.CreateDevice = strings.Split(tag.RepName, "_")[1]
		tag.CreateUser = a.UserName
		tag.UpdateApp = "tag"
		tag.UpdateDevice = strings.Split(tag.RepName, "_")[1]
		tag.UpdateUser = a.UserName
		tag.CreateTime = time
		tag.UpdateTime = time
		tag.RelatedTime = time
		tags = append(tags, tag)
	}
	return tags, nil
}

func (a *allDataDB) getGkillTexts() ([]*reps.Text, error) {
	texts := []*reps.Text{}

	statement := `SELECT ID, Target, Text, Time, RepName FROM "text";`
	rows, err := a.db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all texts from: %w", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		text := &reps.Text{}
		timestr := ""
		err = rows.Scan(&text.ID, &text.TargetID, &text.Text, &timestr, &text.RepName)
		if err != nil {
			err = fmt.Errorf("error at scan rows: %w", err)
			return nil, err
		}
		time, err := time.Parse(TimeLayout, timestr)
		if err != nil {
			err = fmt.Errorf("error at parse time '%s' at %s: %w", timestr, text.ID, err)
			return nil, err
		}
		text.IsDeleted = false
		text.CreateApp = "text"
		text.CreateDevice = strings.Split(text.RepName, "_")[1]
		text.CreateUser = a.UserName
		text.UpdateApp = "text"
		text.UpdateDevice = strings.Split(text.RepName, "_")[1]
		text.UpdateUser = a.UserName
		text.CreateTime = time
		text.UpdateTime = time
		text.RelatedTime = time
		texts = append(texts, text)
	}
	return texts, nil
}

func (a *allDataDB) getGkillTimeiss() ([]*reps.TimeIs, error) {
	rows, err := a.db.Query(`
SELECT 
timeis_start.ID,
timeis_start.Title,
timeis_start.StartTime,
timeis_end.EndTime,
CASE WHEN timeis_end.EndTime IS NULL THEN timeis_start.StartTime ELSE timeis_end.EndTime END AS UpdatedTime,
CASE WHEN timeis_end.EndTime IS NULL THEN timeis_start.RepName ELSE timeis_end.RepName END AS RepName
FROM timeis_start
LEFT OUTER JOIN timeis_end ON timeis_start.ID = timeis_end.StartID
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timeiss := []*reps.TimeIs{}
	for rows.Next() {
		timeis := &reps.TimeIs{}

		startTimeStr, endTimeStr, updateTimeStr := "", sql.NullString{}, ""
		err := rows.Scan(&timeis.ID, &timeis.Title, &startTimeStr, &endTimeStr, &updateTimeStr, &timeis.RepName)
		if err != nil {
			err = fmt.Errorf("error at scan rows at get history: %w", err)
			return nil, err
		}
		timeis.StartTime, err = time.Parse(TimeLayout, startTimeStr)
		if err != nil {
			err = fmt.Errorf("error at parse start time %s: %w", startTimeStr, err)
			return nil, err
		}
		if endTimeStr.Valid {
			timeis.EndTime = &time.Time{}
			*timeis.EndTime, err = time.Parse(TimeLayout, endTimeStr.String)
			if err != nil {
				err = fmt.Errorf("error at parse time: %w", err)
				return nil, err
			}
		}
		timeis.UpdateTime, err = time.Parse(TimeLayout, updateTimeStr)
		if err != nil {
			err = fmt.Errorf("error at parse update time %s: %w", updateTimeStr, err)
			return nil, err
		}
		timeis.IsDeleted = false
		timeis.CreateApp = "timeis"
		timeis.CreateDevice = strings.Split(timeis.RepName, "_")[1]
		timeis.CreateUser = a.UserName
		timeis.UpdateApp = "timeis"
		timeis.UpdateDevice = strings.Split(timeis.RepName, "_")[1]
		timeis.UpdateUser = a.UserName
		timeis.CreateTime = timeis.StartTime
		timeis.UpdateTime = timeis.UpdateTime
		timeiss = append(timeiss, timeis)
	}
	return timeiss, nil
}

func (a *allDataDB) getGkillMis() ([]*reps.Mi, error) {
	mis := []*reps.Mi{}
	statement := `
SELECT 
    Task.TaskID,
    Task.CreatedTime,
    TaskTitleInfo.Title,
    BoardInfo.BoardName,
	CheckStateInfo.IsChecked,
    LimitInfo.LimitTime,
    MiStartInfo.StartTime,
    MiEndInfo.EndTime,

	(SELECT UpdatedTime
    FROM (
        	SELECT datetime(Task.CreatedTime, 'localtime') AS UpdatedTime, Task.RepName AS RepName
			WHERE Task.CreatedTime IS NOT NULL
        UNION
            SELECT datetime(TaskTitleInfo.UpdatedTime, 'localtime') AS UpdatedTime, TaskTitleInfo.RepName AS RepName
			WHERE TaskTitleInfo.UpdatedTime IS NOT NULL
        UNION
            SELECT datetime(BoardInfo.UpdatedTime, 'localtime') AS UpdatedTime, BoardInfo.RepName AS RepName
			WHERE BoardInfo.UpdatedTime IS NOT NULL
        UNION
            SELECT datetime(LimitInfo.UpdatedTime, 'localtime') AS UpdatedTime, LimitInfo.RepName AS RepName
			WHERE LimitInfo.UpdatedTime IS NOT NULL
        UNION
            SELECT datetime(MiStartInfo.UpdatedTime, 'localtime') AS UpdatedTime, MiStartInfo.RepName AS RepName
			WHERE MiStartInfo.UpdatedTime IS NOT NULL
        UNION
            SELECT datetime(MiEndInfo.UpdatedTime, 'localtime') AS UpdatedTime, MiEndInfo.RepName AS RepName
			WHERE MiEndInfo.UpdatedTime IS NOT NULL
        UNION
            SELECT datetime(CheckStateInfo.UpdatedTime, 'localtime') AS UpdatedTime, CheckStateInfo.RepName AS RepName
			WHERE CheckStateInfo.UpdatedTime IS NOT NULL
        ) AS ForUpdateTime
		WHERE ForUpdateTime.UpdatedTime IS NOT NULL
        GROUP BY ForUpdateTime.UpdatedTime, ForUpdateTime.RepName
		ORDER BY ForUpdateTime.UpdatedTime DESC
		LIMIT 1
    ) AS UpdateTime,
    (SELECT RepName
    FROM (
        	SELECT datetime(Task.CreatedTime, 'localtime') AS UpdatedTime, Task.RepName AS RepName
			WHERE Task.CreatedTime IS NOT NULL
        UNION
            SELECT datetime(TaskTitleInfo.UpdatedTime, 'localtime') AS UpdatedTime, TaskTitleInfo.RepName AS RepName
			WHERE TaskTitleInfo.UpdatedTime IS NOT NULL
        UNION
            SELECT datetime(BoardInfo.UpdatedTime, 'localtime') AS UpdatedTime, BoardInfo.RepName AS RepName
			WHERE BoardInfo.UpdatedTime IS NOT NULL
        UNION
            SELECT datetime(LimitInfo.UpdatedTime, 'localtime') AS UpdatedTime, LimitInfo.RepName AS RepName
			WHERE LimitInfo.UpdatedTime IS NOT NULL
        UNION
            SELECT datetime(MiStartInfo.UpdatedTime, 'localtime') AS UpdatedTime, MiStartInfo.RepName AS RepName
			WHERE MiStartInfo.UpdatedTime IS NOT NULL
        UNION
            SELECT datetime(MiEndInfo.UpdatedTime, 'localtime') AS UpdatedTime, MiEndInfo.RepName AS RepName
			WHERE MiEndInfo.UpdatedTime IS NOT NULL
        UNION
            SELECT datetime(CheckStateInfo.UpdatedTime, 'localtime') AS UpdatedTime, CheckStateInfo.RepName AS RepName
			WHERE CheckStateInfo.UpdatedTime IS NOT NULL
        ) AS ForRepName
		WHERE ForRepName.UpdatedTime IS NOT NULL
        GROUP BY ForRepName.UpdatedTime, ForRepName.RepName
		ORDER BY ForRepName.UpdatedTime DESC
		LIMIT 1
	) AS RepName
FROM Task
    LEFT OUTER JOIN TaskTitleInfo ON Task.TaskID = TaskTitleInfo.TaskID 
    LEFT OUTER JOIN BoardInfo ON Task.TaskID = BoardInfo.TaskID 
    LEFT OUTER JOIN CheckStateInfo ON Task.TaskID = CheckStateInfo.TaskID 
    LEFT OUTER JOIN LimitInfo ON Task.TaskID = LimitInfo.TaskID 
    LEFT OUTER JOIN MiStartInfo ON Task.TaskID = MiStartInfo.TaskID 
    LEFT OUTER JOIN MiEndInfo ON Task.TaskID = MiEndInfo.TaskID 
GROUP BY
Task.TaskID,
UpdateTime
HAVING UpdateTime = MAX(datetime(Task.CreatedTime, 'localtime'))
OR UpdateTime = MAX(datetime(TaskTitleInfo.UpdatedTime, 'localtime'))
OR UpdateTime = MAX(datetime(BoardInfo.UpdatedTime, 'localtime'))
OR UpdateTime = MAX(datetime(LimitInfo.UpdatedTime, 'localtime'))
OR UpdateTime = MAX(datetime(MiStartInfo.UpdatedTime, 'localtime'))
OR UpdateTime = MAX(datetime(MiEndInfo.UpdatedTime, 'localtime'))
OR UpdateTime = MAX(datetime(CheckStateInfo.UpdatedTime, 'localtime'))
`
	rows, err := a.db.Query(statement)
	if err != nil {
		err = fmt.Errorf("error at get all tasks: %w", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		mi := &reps.Mi{}
		createdTimeStr, updatedTimeStr := "", ""
		limitTimeStr, startTimeStr, endTimeStr := sql.NullString{}, sql.NullString{}, sql.NullString{}
		isChecked := sql.NullBool{}
		err := rows.Scan(
			&mi.ID,
			&createdTimeStr,
			&mi.Title,
			&mi.BoardName,
			&isChecked,
			&limitTimeStr,
			&startTimeStr,
			&endTimeStr,
			&updatedTimeStr,
			&mi.RepName,
		)
		if err != nil {
			return nil, err
		}

		if isChecked.Valid {
			mi.IsChecked = isChecked.Bool
		}
		mi.CreateTime, err = time.Parse(TimeLayout, createdTimeStr)
		if err != nil {
			err = fmt.Errorf("error at parse create time: %w", err)
			return nil, err
		}
		// 現行バグCheckState初期値0対応のためTimeLayoutをLocalにする
		mi.UpdateTime, err = time.Parse("2006-01-02T15:04:05", strings.ReplaceAll(updatedTimeStr, " ", "T"))
		if err != nil {
			err = fmt.Errorf("error at parse update time: %w", err)
			return nil, err
		}

		if limitTimeStr.Valid {
			mi.LimitTime = &time.Time{}
			*mi.LimitTime, err = time.Parse(TimeLayout, limitTimeStr.String)
			if err != nil {
				err = fmt.Errorf("error at parse limit time: %w", err)
				return nil, err
			}
		}
		if startTimeStr.Valid {
			mi.EstimateStartTime = &time.Time{}
			*mi.EstimateStartTime, err = time.Parse(TimeLayout, startTimeStr.String)
			if err != nil {
				err = fmt.Errorf("error at parse estimate start time: %w", err)
				return nil, err
			}
		}
		if endTimeStr.Valid {
			mi.EstimateEndTime = &time.Time{}
			*mi.EstimateEndTime, err = time.Parse(TimeLayout, endTimeStr.String)
			if err != nil {
				err = fmt.Errorf("error at parse estimate end time: %w", err)
				return nil, err
			}
		}
		mi.IsDeleted = false
		mi.CreateApp = "mi"
		mi.CreateDevice = strings.Split(mi.RepName, "_")[1]
		mi.CreateUser = a.UserName
		mi.UpdateApp = "mi"
		mi.UpdateDevice = strings.Split(mi.RepName, "_")[1]
		mi.UpdateUser = a.UserName
		mi.CreateTime = mi.CreateTime
		mi.UpdateTime = mi.UpdateTime
		mis = append(mis, mi)
	}
	return mis, nil
}

func RepName(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	withoutExt := base[:len(base)-len(ext)]
	return withoutExt
}
