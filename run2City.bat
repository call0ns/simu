# start Parcel generator and city
start "PG" cmd /K "ParcelGenerator\ParcelGenerator -city L_A -timeFactor 60 -bufferMax 100 -workTime 30"
start "City" sampleCity\sampleCity -city L_A
start "PC" ParcelCollector\ParcelCollector -city L_A
start "PG" cmd /K "ParcelGenerator\ParcelGenerator -city L_B -timeFactor 60 -bufferMax 100 -workTime 30"
start "City" sampleCity\sampleCity -city L_B
start "PC" ParcelCollector\ParcelCollector -city L_B
sleep 5
# start trucks
start "Truck" cmd /K "truck\truck -init L_A -id t0 -w 40 -f 60 -ld 1 -s 10"
start "Truck" cmd /K "truck\truck -init L_B -id t1 -w 40 -f 60 -ld 1 -s 10"
