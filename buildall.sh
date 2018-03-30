for i in $(ls); do
	echo $i
	if [ -d "$i" ]; then
		if [ $i != "logs" ]; then
			cd $i
			go build
			cd ..
		fi
	fi
done
