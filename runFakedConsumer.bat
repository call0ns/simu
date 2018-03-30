start "PG" cmd /K "ParcelGenerator\ParcelGenerator"
start "PC" ParcelCollector\ParcelCollector -log 5
start "Consumer" cmd /K "FakedConsumer\FakedConsumer"
