package rrd_test

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"testing"
	"time"

	"github.com/jabdr/rrd"
)

const (
	dbfile    = "/tmp/test.rrd"
	step      = 1
	heartbeat = 2 * step
)

func TestAll(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		c := rrd.NewCreator(dbfile, time.Now(), step)
		c.RRA("AVERAGE", 0.5, 1, 100)
		c.RRA("AVERAGE", 0.5, 5, 100)
		c.DS("cnt", "COUNTER", heartbeat, 0, 100)
		c.DS("g", "GAUGE", heartbeat, 0, 60)
		err := c.Create(true)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Update", func(t *testing.T) {
		u := rrd.NewUpdater(dbfile)
		for i := 0; i < 10; i++ {
			time.Sleep(step * time.Second)
			err := u.Update(time.Now(), i, 1.5*float64(i))
			if err != nil {
				t.Fatal(err)
			}
		}
	})

	t.Run("Update with Cache", func(t *testing.T) {
		u := rrd.NewUpdater(dbfile)
		for i := 10; i < 20; i++ {
			time.Sleep(step * time.Second)
			u.Cache(time.Now(), i, 2*float64(i))
		}
		err := u.Update()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Info", func(t *testing.T) {
		inf, err := rrd.Info(dbfile)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range inf {
			fmt.Printf("%s (%T): %v\n", k, v, v)
		}
	})

	t.Run("Grapher", func(t *testing.T) {
		g := rrd.NewGrapher()
		g.SetTitle("Test")
		g.SetVLabel("some variable")
		g.SetSize(800, 300)
		g.SetWatermark("some watermark")
		g.Def("v1", dbfile, "g", "AVERAGE")
		g.Def("v2", dbfile, "cnt", "AVERAGE")
		g.VDef("max1", "v1,MAXIMUM")
		g.VDef("avg2", "v2,AVERAGE")
		g.Line(1, "v1", "ff0000", "var 1")
		g.Area("v2", "0000ff", "var 2")
		g.GPrintT("max1", "max1 at %c")
		g.GPrint("avg2", "avg2=%lf")
		g.PrintT("max1", "max1 at %c")
		g.Print("avg2", "avg2=%lf")

		now := time.Now()

		i, err := g.SaveGraph("/tmp/test_rrd1.png", now.Add(-20*time.Second), now)
		fmt.Printf("%+v\n", i)
		if err != nil {
			t.Fatal(err)
		}
		i, buf, err := g.Graph(now.Add(-20*time.Second), now)
		fmt.Printf("%+v\n", i)
		if err != nil {
			t.Fatal(err)
		}
		err = ioutil.WriteFile("/tmp/test_rrd2.png", buf, 0666)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Fetch", func(t *testing.T) {
		inf, err := rrd.Info(dbfile)
		if err != nil {
			t.Fatal(err)
		}
		end := time.Unix(int64(inf["last_update"].(uint)), 0)
		start := end.Add(-20 * step * time.Second)
		fmt.Printf("Fetch Params:\n")
		fmt.Printf("Start: %s\n", start)
		fmt.Printf("End: %s\n", end)
		fmt.Printf("Step: %s\n", step*time.Second)
		fetchRes, err := rrd.Fetch(dbfile, "AVERAGE", start, end, step*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer fetchRes.FreeValues()
		fmt.Printf("FetchResult:\n")
		fmt.Printf("Start: %s\n", fetchRes.Start)
		fmt.Printf("End: %s\n", fetchRes.End)
		fmt.Printf("Step: %s\n", fetchRes.Step)
		for _, dsName := range fetchRes.DsNames {
			fmt.Printf("\t%s", dsName)
		}
		fmt.Printf("\n")

		row := 0
		for ti := fetchRes.Start.Add(fetchRes.Step); ti.Before(end) || ti.Equal(end); ti = ti.Add(fetchRes.Step) {
			fmt.Printf("%s / %d", ti, ti.Unix())
			for i := 0; i < len(fetchRes.DsNames); i++ {
				v := fetchRes.ValueAt(i, row)
				fmt.Printf("\t%e", v)
			}
			fmt.Printf("\n")
			row++
		}
	})

	t.Run("Xport", func(t *testing.T) {
		inf, err := rrd.Info(dbfile)
		if err != nil {
			t.Fatal(err)
		}
		end := time.Unix(int64(inf["last_update"].(uint)), 0)
		start := end.Add(-20 * step * time.Second)
		fmt.Printf("Xport Params:\n")
		fmt.Printf("Start: %s\n", start)
		fmt.Printf("End: %s\n", end)
		fmt.Printf("Step: %s\n", step*time.Second)

		e := rrd.NewExporter()
		e.Def("def1", dbfile, "cnt", "AVERAGE")
		e.Def("def2", dbfile, "g", "AVERAGE")
		e.CDef("vdef1", "def1,def2,+")
		e.XportDef("def1", "cnt")
		e.XportDef("def2", "g")
		e.XportDef("vdef1", "sum")

		xportRes, err := e.Xport(start, end, step*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer xportRes.FreeValues()
		fmt.Printf("XportResult:\n")
		fmt.Printf("Start: %s\n", xportRes.Start)
		fmt.Printf("End: %s\n", xportRes.End)
		fmt.Printf("Step: %s\n", xportRes.Step)
		for _, legend := range xportRes.Legends {
			fmt.Printf("\t%s", legend)
		}
		fmt.Printf("\n")

		row := 0
		for ti := xportRes.Start.Add(xportRes.Step); ti.Before(end) || ti.Equal(end); ti = ti.Add(xportRes.Step) {
			fmt.Printf("%s / %d", ti, ti.Unix())
			for i := 0; i < len(xportRes.Legends); i++ {
				v := xportRes.ValueAt(i, row)
				fmt.Printf("\t%e", v)
			}
			fmt.Printf("\n")
			row++
		}
	})
/*
	t.Run("StreamDump", func(t *testing.T) {
		fmt.Println("Stream Dump Test Output:")
		ct := make(chan bool)
		rrdRows := make(chan *rrd.RrdDumpRow, 100)
		if err := rrd.StreamDump(dbfile, "AVERAGE", rrdRows, ct); err != nil {
			t.Fatal(err)
		}
		timeout := time.After(time.Second * 30)
	fl:
		for {
			select {
			case row := <-rrdRows:
				if row != nil {
					vals := make([]string, len(row.Values))
					for i, c := range row.Values {
						vals[i] = strconv.FormatFloat(c, 'f', 3, 64)
					}
					fmt.Printf("%d-%02d-%02dT%02d:%02d:%02d-00:00:  %s\n",
						row.Time.Year(), row.Time.Month(), row.Time.Day(),
						row.Time.Hour(), row.Time.Minute(), row.Time.Second(), strings.Join(vals, ", "))
				} else {
					fmt.Println("finished stream")
					break fl
				}
			case <-timeout:
				t.Fatal("Timeout")
			}
		}
	})

	t.Run("StreamDump Memory", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode")
		}
		fmt.Println("Stream Dump Memory Test:")
		printMemUsage()
		for tst := 0; tst < 100000; tst++ {
			ct := make(chan struct{})
			rrdRows := make(chan *rrd.RrdDumpRow, 100)
			if done, err := rrd.StreamDump(dbfile, "AVERAGE", rrdRows, ct); err != nil {
				t.Fatal(err)
			}
			timeout := time.After(time.Second * 30)
		fl:
			for {
				select {
				case row := <-rrdRows:
					if row == nil {
						break fl
					}
				case <-timeout:
					t.Fatal("Timeout")
				}
			}
			runtime.GC()
			if tst%1000 == 0 {
				fmt.Printf("Run test %d\n", tst)
				printMemUsage()
			}
		}
		fmt.Println("Completed memory bulk test")
		printMemUsage()
	})*/

}

func ExampleCreator_DS() {
	c := &rrd.Creator{}

	// Add a normal data source, i.e. one of GAUGE, COUNTER, DERIVE and ABSOLUTE:
	c.DS("regular_ds", "DERIVE",
		900, /* heartbeat */
		0,   /* min */
		"U" /* max */)

	// Add a computed
	c.DS("computed_ds", "COMPUTE",
		"regular_ds,8,*" /* RPN expression */)
}

func ExampleCreator_RRA() {
	c := &rrd.Creator{}

	// Add a normal consolidation function, i.e. one of MIN, MAX, AVERAGE and LAST:
	c.RRA("AVERAGE",
		0.3, /* xff */
		5,   /* steps */
		1200 /* rows */)

	// Add aberrant behavior detection:
	c.RRA("HWPREDICT",
		1200, /* rows */
		0.4,  /* alpha */
		0.5,  /* beta */
		288 /* seasonal period */)
}

func printMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
