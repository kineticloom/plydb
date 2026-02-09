package main

import (
	"fmt"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/ypt/experiment-nexus/sqlwalk"
)

func main() {
	rawQuery2 := `with combined as (
	  select * from db.accounts.organizations o
	  join files.organization_priorities oca
	    on o.id = oca.organization_id
	  join files.organization_sizes oca2
	    on o.id = oca2.organization_id
	)
	select * from combined
	where priority = 'red'
	and revenue_amount_limit <= 100000`
	result, err := pg_query.Parse(rawQuery2)
	if err != nil {
		panic(err)
	}
	lr := sqlwalk.Extract(result)

	fmt.Println("Tables:")
	for _, t := range lr.Tables {
		fmt.Printf("  catalog=%q schema=%q name=%q alias=%q\n", t.Catalog, t.Schema, t.Name, t.Alias)
	}
}
