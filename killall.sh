echo $1

for ps in truc Parc sam city; do
	ps | grep $ps | while read line
	do
		echo $line
		for i in $line; do
			kill $i
			break
		done
	done
done
