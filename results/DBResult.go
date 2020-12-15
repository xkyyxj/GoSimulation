package results

type DBResult interface {
	query(where string)
	insert()
	update()
	delete()
}
