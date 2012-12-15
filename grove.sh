#!/bin/bash
### Startup Script for Grove ###
# To enable: (on debian)
# ln -s /etc/init.d/grove /path/to/this/script/grove.sh
#
# To use:
# sudo service grove {start|stop|restart|status|check}
#

GROVE=grove
LOG=/dev/null

PID=$(pidof -o %PPID $GROVE)

start()
{
	if [ -z $PID ]; then
		$GROVE 2>&1 >> $LOG &
	fi
}
stop()
{
	if [ ! -z $PID ]; then
		kill $PID
	fi
}
restart()
{
	stop
	start
}
status()
{
	echo -n "* Grove is "
	if [ -z $PID ]; then
		echo "not running."
		exit 1
	else
		echo "running."
		exit 0
	fi
}
check()
{
	status > /dev/null
	if [ $? == 1 ]; then
		start
	fi
}

case "$1" in
	"start" )
		start
		;;
	"stop" )
		stop
		;;
	"restart" )
		restart
		;;
	"status" )
		status
		;;
	"check" )
		check
		;;
	* )
		echo "usage: $0 {start|stop|restart|status|check}"
esac