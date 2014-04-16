package main

import "github.com/mattn/termbox-go"
import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type death_disk struct {
	name     string
	pass     string
	percent  uint64
	timeLeft time.Duration
}

var (
	flagDisks     = flag.String("drives", "", "Comma delimited list of drives to destroy")
	flagBlockSize = flag.Int("bs", 512, "Write block size in KB")
	flagPasses    = flag.Int("passes", 0, "How many passes to make on the disk.  Zero nulls disk")
	disks         []string
	dieDisks      []death_disk
	quit          bool
	termClosed    bool
	bs            uint64
	passes        uint64
)

func init() {
	flag.Parse()
	if *flagDisks == "" {
		fmt.Printf("You must select a drive for destruction\n")
		os.Exit(-1)
	}
	disks = strings.Split(*flagDisks, ",")
	if len(disks) <= 0 {
		fmt.Printf("You must select a drive for destruction\n")
		os.Exit(-1)
	}
	for _, x := range disks {
		x := death_disk{x, "FIRST", 0, 0}
		dieDisks = append(dieDisks, x)
	}
	bs = uint64(*flagBlockSize) * uint64(1024)
	passes = uint64(*flagPasses)
	if passes == 0 {
		passes = 1
	}
	quit = false
	termClosed = false
}

func main() {
	err := verify_drives(dieDisks)
	if err != nil {
		fmt.Printf("Could not open /dev/%s for destruction (%s)\n", *flagDisks, err)
		os.Exit(-1)
	}

	fmt.Printf("About to destroy the following drives:\n")
	for _, x := range dieDisks {
		fmt.Printf("\t%s\n", x.name)
	}
	var answer string
	fmt.Printf("Are you SURE <yes/no> ")
	fmt.Scanf("%s", &answer)
	if answer != "yes" {
		fmt.Printf("I got a %s.  I need a \"yes\"\n", answer)
		os.Exit(-1)
	}

	go draw_updates()
	go destroyAll()
	for quit != true {
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Printf("\n\n")
	for termClosed != true {
		time.Sleep(10 * time.Millisecond)
	}
}

func verify_drives(drives []death_disk) error {
	for i := range drives {
		_, err := os.Stat(drives[i].name)
		if err != nil {
			return err
		}
		err = os.Chmod(drives[i].name, 0x060)
		if err != nil {
			return err
		}

		fi, err := os.OpenFile(drives[i].name, os.O_RDWR, 0666)
		if err != nil {
			return err
		}
		defer fi.Close()
	}
	return nil
}

func setBlock(block []byte, size uint64, stage uint8) error {
	switch stage {
	case 1:
		for i := range block {
			block[i] = 0xff
		}
	case 2:
		for i := range block {
			block[i] = 0x55
		}
	case 3:
		for i := range block {
			block[i] = 0xAA
		}
	case 0:
	default:
		for i := range block {
			block[i] = 0
		}
	}
	return nil
}

func destroyOne(index int) {
	fi, err := os.OpenFile(dieDisks[index].name, os.O_RDWR, 0666)
	if err != nil {
		dieDisks[index].pass = "ERROR"
		return
	}

	defer fi.Close()

	//determine size of disk
	fileSize, err := fi.Seek(0, 2)
	if err != nil {
		dieDisks[index].pass = "ERROR"
		return
	}
	_, err = fi.Seek(0, 0)
	if err != nil {
		dieDisks[index].pass = "ERROR"
		return
	}
	totalBytesToWrite := uint64(uint64(fileSize) * passes)
	totalBytesWritten := uint64(0)
	timeSampleBytesWritten := uint64(0)
	t := time.Now()
	startTime := t

	block := make([]byte, bs, bs)
	for i := uint64(0); i < passes; i++ {
		switch i % 4 {
		case 0:
			dieDisks[index].pass = "ZERO"
		case 1:
			dieDisks[index].pass = "FF"
		case 2:
			dieDisks[index].pass = "01"
		case 3:
			dieDisks[index].pass = "10"
		default:
			dieDisks[index].pass = "ZERO"
		}
		setBlock(block, bs, uint8(i%4))
		for i := int64(0); i <= fileSize; i += int64(bs) {
			n, err := fi.WriteAt(block, i)
			fi.Sync()
			totalBytesWritten += uint64(bs)
			timeSampleBytesWritten += uint64(bs)

			if i > 0 && err != nil {
				dieDisks[index].percent = 100
				break
			} else if n == 0 && err != nil {
				dieDisks[index].pass = "ERROR"
				return
			}
			dieDisks[index].percent = uint64((i * 100) / fileSize)
			timePassed := time.Since(t)
			if timePassed >= (3 * time.Second) {
				t = time.Now()

				bytesLeft := totalBytesToWrite - totalBytesWritten
				totalSeconds := uint64(time.Since(startTime).Seconds())
				avgBytesPerSecond := totalBytesWritten / totalSeconds
				secondsLeft := bytesLeft / avgBytesPerSecond
				dieDisks[index].timeLeft = time.Duration(time.Second * time.Duration(secondsLeft))

				timeSampleBytesWritten = 0
			}
		}
	}
	dieDisks[index].pass = "DONE"
}

func destroyAll() {
	//check if disk exists

	//attempt to open the file handle

	//set initial state
	for i := range dieDisks {
		dieDisks[i].pass = "FIRST"
		go destroyOne(i)
	}

	//destroy it
	for quit != true {
		time.Sleep(100 * time.Millisecond)
		notDone := false
		for i := range dieDisks {
			if dieDisks[i].percent < 100 {
				notDone = true
			}
		}
		if notDone == false {
			quit = true
			break
		}
	}
}

func init_msg_lines() []string {
	var msg_lines []string
	msg_lines = append(msg_lines, fmt.Sprintf("Disk Destroyer PRO TURBO 9000 edition (with social integration)"))
	msg_lines = append(msg_lines, fmt.Sprintf("Press ESC to quit"))
	msg_lines = append(msg_lines, fmt.Sprintf(""))
	for _, v := range dieDisks {
		msg_lines = append(msg_lines, fmt.Sprintf("%s %d%% %s ", v.name, v.percent, v.pass))
	}
	return msg_lines
}

func prep_msg_lines(msg_lines []string) []string {
	w, h := termbox.Size()
	for i := range dieDisks {
		if (i + 3) > h {
			break
		}
		if dieDisks[i].percent >= uint64(100) {
			msg_lines[i+3] = fmt.Sprintf("DONE WITH %s", dieDisks[i].name)
			time.Sleep(100 * time.Millisecond)
		} else {
			msg_lines[i+3] = fmt.Sprintf("%s %d%% %v %s ", dieDisks[i].name, dieDisks[i].percent, dieDisks[i].timeLeft, dieDisks[i].pass)
			time.Sleep(100 * time.Millisecond)
			/* figure out space for progress bar */
			total := w - len(msg_lines[i+3])
			msg_len := len(msg_lines[i+3])

			/* set progress */
			to_set := uint64((uint64(total) * dieDisks[i].percent) / 100)
			for j := uint64(msg_len); (j < uint64(w)) && (j < uint64(to_set)); j++ {
				msg_lines[i+3] += "#"
			}
			/* clear out the progress bar */
			for j := len(msg_lines[i+3]); j < total; j++ {
				msg_lines[i+3] += " "
			}
		}
	}
	return msg_lines
}

func draw_updates() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}

	event_queue := make(chan termbox.Event)
	go func() {
		for {
			event_queue <- termbox.PollEvent()
		}
	}()
	msg_lines := init_msg_lines()
	draw(msg_lines)
loop:
	for {
		select {
		case ev := <-event_queue:
			if ev.Type == termbox.EventKey && ev.Key == termbox.KeyEsc {
				quit = true
				break loop
			}
		default:
			msg_lines = prep_msg_lines(msg_lines)
			draw(msg_lines)
			if quit {
				break
			}
		}
	}
	termbox.Close()
	termClosed = true
}

func draw(msg_lines []string) {

	w, h := termbox.Size()
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	//set the message
	for y := 0; y < len(msg_lines) && y < h; y++ {
		i := 0
		for _, s := range msg_lines[y] {
			if i > w {
				break
			} else {
				i++
			}
			termbox.SetCell(i, y, s, termbox.ColorDefault, termbox.ColorDefault)
		}
	}
	termbox.Flush()
}
