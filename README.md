# bmpmon
Just a go-playground after the [sarpi](https://sarpi.penthux.net/) project was found...

```
                  :::::::                      root@sarpi
            :::::::::::::::::::                ----------
         :::::::::::::::::::::::::             OS: Slackware 15.0 aarch64 (post 15.0 -current) aarch64
       ::::::::cllcccccllllllll::::::          Host: Raspberry Pi 3 Model B Rev 1.2
    :::::::::lc               dc:::::::        Kernel: 6.6.31-v8-sarpi3_64
   ::::::::cl   clllccllll    oc:::::::::      Uptime: 6 days, 20 hours, 14 mins
  :::::::::o   lc::::::::co   oc::::::::::     Packages: 629 (pkgtool)
 ::::::::::o    cccclc:::::clcc::::::::::::    Shell: bash 5.2.26
 :::::::::::lc        cclccclc:::::::::::::    Terminal: /dev/pts/0
::::::::::::::lcclcc          lc::::::::::::   CPU: (4) @ 1.200GHz
::::::::::cclcc:::::lccclc     oc:::::::::::   Memory: 110MiB / 909MiB
::::::::::o    l::::::::::l    lc:::::::::::
 :::::cll:o     clcllcccll     o:::::::::::
 :::::occ:o                  clc:::::::::::
  ::::ocl:ccslclccclclccclclc:::::::::::::
   :::oclcccccccccccccllllllllllllll:::::
    ::lcc1lcccccccccccccccccccccccco::::
      ::::::::::::::::::::::::::::::::
        ::::::::::::::::::::::::::::
           ::::::::::::::::::::::
                ::::::::::::
```
## points of interest
1. [`sensor.go`](https://github.com/dandriano/bmpmon/blob/master/sensor.go) - service-like wrapper around existing Bosch BMP Sensor library (also, early zyre-composited [variant](https://github.com/dandriano/bmpmon/blob/dd940aabfe7a73b01b2483e1b20ce303d1dd9b96/sensor.go#L55))
2. [`storage.go`](https://github.com/dandriano/bmpmon/blob/master/storage.go) - minimal sqlite-based storage (alas, with out unitofwork but with buffer example)
3. [`main.go`](https://github.com/dandriano/bmpmon/blob/master/main.go#L45-L111) - html-view builded with [`go-echarts`](https://github.com/go-echarts/go-echarts) (also, a real-time alike [implementation](https://github.com/go-echarts/statsview))
