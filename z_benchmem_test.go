// Copyright 2016 Tamás Gulácsi, Valentin Kuznetsov. All rights reserved.
// Use of this source code is governed by The MIT License
// found in the accompanying LICENSE file.

package ora_test

import (
	"runtime"
	"testing"

	"gopkg.in/rana/ora.v3"
)

func BenchmarkPrepare(b *testing.B) {
	rows, err := testDb.Query("SELECT A.object_name from all_objects A")
	if err != nil {
		b.Fatal(err)
	}
	b.StopTimer()
	rows.Close()
}

func BenchmarkIter(b *testing.B) {
	b.StopTimer()
	rows, err := testDb.Query("SELECT A.object_name from all_objects A")
	if err != nil {
		b.Fatal(err)
	}
	defer rows.Close()
	b.StartTimer()
	i := 0
	for rows.Next() && i < b.N {
		i++
	}
	b.SetBytes(int64(i))
}

// BenchmarkMemory usage for querying rows.
//
// go test -c && ./ora.v3.test -test.run=^$ -test.bench=Memory -test.memprofilerate=1 -test.memprofile=/tmp/mem.prof && go tool pprof --alloc_space ora.v3.test /tmp/mem.prof
func TestMemoryNumString(t *testing.T) {
	n := 1000
	benchMem(t, n, 1360, `SELECT
		TO_NUMBER('123456789012345678') bn01
		, TO_NUMBER('223456789012345678') bn02
		, TO_NUMBER('323456789012345678') bn03
		, TO_NUMBER('423456789012345678') bn04
		, TO_NUMBER('523456789012345678') bn05
		, TO_NUMBER('623456789012345678') bn06
		, TO_NUMBER('723456789012345678') bn07
		, TO_NUMBER('823456789012345678') bn08
		, TO_NUMBER('923456789012345678') bn09
		, TO_NUMBER('023456789012345678') bn10
	FROM ALL_OBJECTS B, all_objects A WHERE ROWNUM <= :1`)
}
func TestMemoryNumStringI64(t *testing.T) {
	ora.Cfg().Env.StmtCfg.Rset.SetNumberBigInt(ora.I64)
	ora.Cfg().Env.StmtCfg.Rset.SetNumberBigFloat(ora.I64)
	defer func() {
		ora.Cfg().Env.StmtCfg.Rset.SetNumberBigInt(ora.N)
		ora.Cfg().Env.StmtCfg.Rset.SetNumberBigFloat(ora.N)
	}()
	n := 1000
	benchMem(t, n, 1352, `SELECT
		TO_NUMBER('123456789012345678') bn01
		, TO_NUMBER('223456789012345678') bn02
		, TO_NUMBER('323456789012345678') bn03
		, TO_NUMBER('423456789012345678') bn04
		, TO_NUMBER('523456789012345678') bn05
		, TO_NUMBER('623456789012345678') bn06
		, TO_NUMBER('723456789012345678') bn07
		, TO_NUMBER('823456789012345678') bn08
		, TO_NUMBER('923456789012345678') bn09
		, TO_NUMBER('023456789012345678') bn10
	FROM ALL_OBJECTS B, all_objects A WHERE ROWNUM <= :1`)
}

func TestMemoryString(t *testing.T) {
	n := 1000
	benchMem(t, n, 1432, `SELECT
		'123456789012345678' bs01
		, '223456789012345678' bs02
		, '323456789012345678' bs03
		, '423456789012345678' bs04
		, '523456789012345678' bs05
		, '623456789012345678' bs06
		, '723456789012345678' bs07
		, '823456789012345678' bs08
		, '923456789012345678' bs09
		, '023456789012345678' bs10
	FROM ALL_OBJECTS B, all_objects A WHERE ROWNUM <= :1`)
}

func benchMem(tb testing.TB, n int, maxBytesPerRun uint64, qry string) {
	columns, err := ora.DescribeQuery(testDb, qry)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Logf("columns: %#v", columns)

	cols := make([]string, len(columns))
	for i, c := range columns {
		cols[i] = c.Name
	}
	args := []interface{}{int64(n)}

	type Record map[string]interface{}

	execute := func(qry string, cols []string, args ...interface{}) []Record {
		var out []Record

		rows, err := testDb.Query(qry, args...)
		if err != nil {
			tb.Fatalf("ERROR: DB.Query, query='%s' args='%v' error=%v", qry, args, err)
		}
		defer rows.Close()

		count := len(cols)
		vals := make([]interface{}, count)
		valPtrs := make([]interface{}, count)
		for i, _ := range cols {
			valPtrs[i] = &vals[i]
		}
		// loop over rows
		for rows.Next() {
			err := rows.Scan(valPtrs...)
			//        err := rows.Scan(vals...)
			if err != nil {
				tb.Fatalf("ERROR: rows.Scan, dest='%v', error=%v", vals, err)
			}
			rec := make(Record)
			length := 0
			for i, _ := range cols {
				rec[cols[i]] = vals[i]
				if s, ok := vals[i].(string); ok {
					length += len(s)
				}
			}
			out = append(out, rec)
			if len(out) == 1 {
				tb.Logf("One record's length: %d", length)
			}
		}
		if err = rows.Err(); err != nil {
			tb.Fatal(err)
		}
		return out
	}

	var ostat, nstat runtime.MemStats
	runtime.ReadMemStats(&ostat)
	results := execute(qry, cols, args...)
	runtime.ReadMemStats(&nstat)
	d := nstat.TotalAlloc - ostat.TotalAlloc
	tb.Logf("nres=%d, allocated %d bytes\n", len(results), d)
	maxBytes := maxBytesPerRun * uint64(n)
	if maxBytes > 0 && d > maxBytes {
		tb.Errorf("nres=%d, allocated %d bytes (max: %d)", len(results), d, maxBytes)
	}
	ostat = nstat
}