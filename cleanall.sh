for i in $(ls); do
	echo $i
	if [ -d "$i" ]; then
		if [ $i != "logs" ]; then
			cd $i
			go clean
			cd ..
		else
			rm -rf $i/*.log
		fi
	fi
done
