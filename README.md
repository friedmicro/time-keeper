# TimeKeeper

A self hostable time tracking app for people who want something to keep track of their time.

## Why
I have always had a lot of unstructured work in my life which has not inherantly been an issue for me. But, it has encouraged me to put systems in place for tracking my time honestly for myself and for any employers. Having something in place forces me to make sure everything is done like it should for the day and stops me from working until the sun comes up the next morning (yes I have done that before, but it was wonderful bit of code).

In the past I have used some timetracking software, but most of these are heavilly geared towards billing as a freelancer or consultant and I found them all to lack some or all of the above: the fine grained control I wanted on an API, the ability to do analysis like I want, the ability to restrict activity on a system, privacy\self hosted, support for automations, or a simple UI. So being tired of the solutions available not matching my needs I opted to write my own.

I wanted a time tracking system that let's me lock myself out of my phone without resorting to a random app the locks during x hours of time and I wanted something would also lock me out of certain Linux applications. The reason is fairly simple, if the work is keeping my skills sharp and building projects the most productive option for myself is to simply lock myself out of distractions that to me are lower priority, but the only way to do that would be if I could control the API because otherwise I am depending on multiple software applications which do not communicate with each other or do so fairly limited.

Disclaimer\background everyone uses this API in the following manner has agreed to it and several have asked me. This same API also can do things like notify a partner when I'm working so that they don't walk into the den and start chatting then pause...think for a second and be like my bad are you working? This is a much simpler solution because a push notification automatically goes to their phone, they have an API they can poll (if they're an engineer), or I can do something low tech like light up a small sign with an IR burster. And honestly, it does miracles for relationships because it just clearly puts that info in front to say...hey I love you, but I'm working...I might be a second to repond to a text and maybe we can have that lengthy chat in a few hours?

I also don't like the idea of my time for personal projects that may not be public yet being stored on someone's servers that can leak. I have in the past run my own company and written software which I did not want public I was working on at the time. To me the risk of a leak and that dropping every personal project I am working on into the public eye or disclosing the exact hours I worked on them is a major reason to run my own system for this. And I don't the reality that if I loose an internet connection or the service goes offline then I can't log what I've been up to.

It's also hard when you use a hosted service to cleanly and securely do automations because functionally you have to figure out a way for service x to call back into your network or you have to poll the service (which often is not possible because a lot of these only give you a webhook). For example I want to turn off my monitors on my PC in the den if I am in the living room because I have 4 of them and they idle draw a fair amount of power (sleep is off due to issue on my Linux install). And it allows little things like for example there is a entity in home assistant that is updated with my working state that allows automations to trigger different paths if I am working or not (ex disable the automatic light controls in a room).

Having a system like this also gives you the option to analyze your work for patterns. For example...do I typically work at night or during the day? What projects do I spend the most time on? While I'm not personally doing this right now it is something that I have done in the past when I was using pen\paper or old time keeping software so even though it has been a few years since I did that it is entirely possible I opt to do it again.

As for the UI...all I wanted on every time keeping program UI was a start\stop and a way to see my time. Most threw a login page, a prompt for the project (which this does support, but its not a required field), and just generally felt extremely formal\cluttered as they were a solution being offered rather than a single program for a user. So that's what this is...a very simple web UI, GTK client, and waybar binary that just show the info, render it cleanly, and let you start\stop. And on the backend so it's not always on your mind are the automations themselves for homeassistant and things like letting people know you are working.

## How it works
On the default deployment which is a high availability deployment with the API and frontend in the same container what happens is you go to the website, hit start to start working, and then hit stop to stop working. The frontend summaries page requests an API that will read the sessions since Monday and return these back on day by day basis (see the Main API section for the exact message contract). And this in itself is the core of it and might be enough for most people using this, the other services here are just nice to haves.

The current api is used by home assistant to set a custom entity, the frontend to handle currently running sessions, and the GTK app for the same purpose.

The week_times API takes the number of hours worked that week after reading the sessions from the DB and returns: a total, the amount per day, whether you are done working (days * 8), actively working, and if a break is due (at least 6 hours worked today). This API is used by the state service to return if you are working, whether you are free (you are not working and you have 8 hours per day met), whether you have worked less than 6 hours that day, or if you have worked 6 hours (optionally useable to be a...yeah maybe grabbing lunch isn't a terrible idea if we haven't already). Do note these times are configurable. Athena also uses this API but explicitly checks for the break flag so that fun applications can be opened on a break. Fun note...the break flag sets to true for a few hours in the evening so that a game can be opened at 1am if you happen to be awake.

The state service is used by the notification service as well as mobile devices running things like Shortcut or Tasker. While the API of the time keeper could just directly accessed there is a lot of logic here that is configurable with respect to the person's who this API is for. For example you might think that the limit on the states should be much lower like say 2 hours for coding projects or 1 hour for studying for an exam. Without this service you would have to configure this on each client and that includes clients you may not control (ex a partner's).

On the DB there is a single table called sessions (there is no database normalization) which includes the start, stop, and category. The start endpoint creates a new entry and stop endpoint adds the stop. This is how the actual times are calculated. Due to a bug these are stored in UTC and a small block of code calculates\fixes the broken offset to the current timezone, this is not ideal, but migrating the existing deployments was not optimal.

The GTK client uses the same endpoints as the frontend. Waybar uses the week_times endpoint because it's the most granular data. The /day endpoint is an API only integration which just calls the same code as start or stop depending on if there is a session open, I keep it around so an old bash script keeps working on my laptop because I use the terminal more on that machine. Likewise add_8_hours and add_time are API only and exist as overrides for things like say forgetting to start a timer or adding something like a vacation.

The backend services are entirely stateless because only the main API is actually calling the DB and nothing with the exception of the cellnotify is being kept in memory (and even that is just computed again at start so a crash is a nonissue). The low memory footprint also makes high availability fairly trivial so for something that you want to never go down this is perfect.

## APIs

### Main API:
| Method | Path            | Description                              | Example Response |
|--------|-----------------|------------------------------------------|--------|
| GET    | `/api/week_times` | Weekly per-day time totals + quotas     | {"break_met":false,"days":{"0":{"date":"2026-10-25","hours":8.06},"1":{"date":"2026-10-27","hours":1.36}},"quota_met":false,"today_hours":1.36,"totals":9.42} |
| GET    | `/api/week`      | Weekly summary JSON (includes categories) | {"categories":[{"date":"2026-10-27","category":"","total":"1h 22m 53s"},{"date":"2026-10-25","category":"uncategorized","total":"8h 3m 48s"}],"days":[{"date":"2026-10-27","total":"1h 22m 53s"},{"date":"2026-10-25","total":"8h 3m 48s"}],"week_total":"9h 26m 41s"} |
| GET    | `/api/current`  | Current run state (`running`, `started_at`) | {"running":true,"started_at":"2026-10-27T18:41:35Z"} |
| POST   | `/api/start`    | Start a new run (optional `category`)     | |
| POST   | `/api/stop`     | Stop the current run          | |
| POST   | `/api/add_8_hours` | Add 8h to the hours worked | |
| POST   | `/api/add_time` | Add arbitrary hours to the weekly time | |

### State API
| Method | Path            | Description                              | Example Response |
|--------|-----------------|------------------------------------------|--------|
| GET    | `/api/state` | A high level state usable in automations     | {"message":"Lucia is working","state":"lucia_work"} |

### Waybar API

Returns a JSON body when called that matches\conforms to the expected output of waybar for a custom data element in the waybar project (https://github.com/alexays/waybar). By default this would match:
```
{
"text": "Text displayed in waybar...for us this is the number of hours worked in the week so far",
"alt": "Alternative text if you opt to set this in the config...for us this is the number of hours worked in that day",
"tooltip": "The text that renders when you hover the element...for us this renders a a breakdown of hours this week, hours today, if you need a break, or if you are done for the day",
"class": "Used for syling...for us a class is set depending on if you are done with work for the day your waybar styling will still need to use this class however",
"percentage": "A percent used for something like a loading bar...currently we always set 100"
}
```

So the output will look like this:

```
{
"text":"9.54",
"alt":1.47,
"tooltip":"Totals: 9.54 hours\nHours today: 1.47\nQuota: not met\nBreak: not met",
"class":"quota-unmet",
"percentage":100}
```

## Build\run\layout

Simple run `make`. Everything will build from a simple make file. For a production deployment you can optionally use the Dockerfile at the root and in the various projects.

Of note the following docker files exist and are the services you would install in a cluster:
root level: API\frontend
cellnotify: cell notifications
state: a high level state API for automation

Some extra utilities you might like:
core: a C++ library for writing client integrations, currently used by the GTK client
gtk: a simple GTK application that lets you start\stop the time and see the current amount of time worked since your last break

In addition to this you will need to configure any and all environment variables to your needs.


## Environment variables

### api/

| Variable | Default | Purpose |
|----------|---------|---------|
| `DB_HOST` | `localhost` | MySQL host |
| `DB_USER` | `root` | MySQL user |
| `DB_PASSWORD` | *(empty)* | MySQL password |
| `DB_NAME` | *(empty)* | MySQL database name |
| `PORT` | `5006` | HTTP listen port |
| `SERPENTS_CLIENT_IP` | *(empty)* | If a device on this IP starts or stops work trigger a webhook |
| `SERPENTS_WEBHOOK_URL` | *(empty)* | The webhook to be triggered |
| `TIMEZONE` | `America/Los_Angeles` | Wall-clock timezone for day/week math |

### state/

| Variable | Default | Purpose |
|----------|---------|---------|
| `TIME_STATE_LISTEN` | `:5000` | HTTP listen address |
| `TIMEKEEPER_API` | `http://time.lucia.friedmicro-lab.org` | Base URL of the TimeKeeper API |
| `STATE_WORK` | `lucia_work` | State value reported while working |
| `STATE_LOW_HOURS` | `lucia_low_hours` | State value when low on hours |
| `STATE_NOT_WORK` | `lucia_not_work` | State value when not working |
| `STATE_FREE` | `lucia_free` | State value when quota/break fully met |
| `STATE_WORK_TEXT` | `Lucia is working` | Human text for the working state |
| `STATE_LOW_HOURS_TEXT` | `Lucia is low on hours` | Human text for the low-hours state |
| `STATE_NOT_WORK_TEXT` | `Lucia is not working` | Human text for the not-working state |
| `STATE_FREE_TEXT` | `Lucia is free for day` | Human text for the free state |
| `WORK_HOURS` | `8` | Daily work-hour target |
| `WORK_LOW_HOURS` | `6` | Daily low-hours threshold |
| `TIMEZONE` | `America/Los_Angeles` | Wall-clock timezone for day math |

### cellnotify/

| Variable | Default | Purpose |
|----------|---------|---------|
| `TIMEKEEPER_API` | *(empty)* | Base URL of the TimeKeeper API |
| `TIMEZONE` | `America/Los_Angeles` | Wall-clock timezone for day math |
| `PUSHCUT_URL` | *(empty)* | Pushcut notification URL to POST state changes to |
| `STATE_WORK` | `lucia_work` | State value reported while working that is sent to the webhook |
| `STATE_LOW_HOURS` | `lucia_low_hours` | State value when low on hours that is sent to the webhook |
| `STATE_NOT_WORK` | `lucia_not_work` | State value when not working that is sent to the webhook |
| `STATE_FREE` | `lucia_free` | State value when quota/break fully met that is sent to the webhook |
| `STATE_WORK_TEXT` | `Lucia is working` | Human text for the working state |
| `STATE_LOW_HOURS_TEXT` | `Lucia is low on hours` | Human text for the low-hours state |
| `STATE_NOT_WORK_TEXT` | `Lucia is not working` | Human text for the not-working state |
| `STATE_FREE_TEXT` | `Lucia is free for day` | Human text for the free state |
| `WORK_HOURS` | `8` | Daily work-hour target |
| `WORK_LOW_HOURS` | `6` | Daily low-hours threshold |

### gtk/ (timekeeper-gtk)

| Variable | Default | Purpose |
|----------|---------|---------|
| `TIMEKEEPER_API` | `http://time.lucia.friedmicro-lab.org` | Base URL of the TimeKeeper API |

### notificationhandler/

| Variable | Default | Purpose |
|----------|---------|---------|
| `TIMEKEEPER_API` | `http://time.lucia.friedmicro-lab.org` | Base URL of the TimeKeeper API (polled for working state) |
| `DISTRACTION_KILLER_BINARY` | `Discord` | Application binary to kill while working |
| `DISTRACTION_KILLER_VOICE` | `WEBRTC VoiceEngine` | PipeWire voice target to disconnect while working |
| `DISTRACTION_KILLER_INTERVAL` | `120` | Poll interval in seconds (min 10) |

### timewaybar/

timewaybar reads no environment variables. It loads `timewaybar.conf` from its
working directory (keys: `url`/`api_url` for the API base, `state_file` for the
cached-state JSON path).

### Misc application behavior notes and design decisions

- Active work sessions are calculated at API call not by a background service meaning a restart of the service or a disconnect will not stop them
- If the timewaybar binary fails to make a request then it will render the last connected state
- Most services are designed to run in high availability and are designed to be stateless, this is something this is personally run on a k8s cluster. The only service that does not support HA on the cluster is the cellnotify service, but that is a stateless service so will simply send the current state twice if it crashes.
- The use of go without a linked c lib means Linux distro compatibility should be pretty wide for everything that is expected to run server side.
- The simplified state API exists mostly for low code integrations for thinks like iOS shortcuts or Android Tasker and this is the primary usecase.
- MariaDB is used as it has higher performance on simple table architectures and to keep everything simple this project is effectively all data in some table. Only session starts\stops are recorded.
- Working beyond 40 hours will add that additional time to next week's time worked.
- While the API only returns back one week the DB is simple enough adding a query to analyze your past work is fairly trivial.

### Integrations
- Home assistant: If `SERPENTS_CLIENT_IP` and `SERPENTS_WEBHOOK_URL` are set then if a device with the specified IP starts or stops a work session the respective webhook will be called. This can be and is currently used for automations. https://github.com/home-assistant
- Athena: The time tracking API is used by Athena so if you have set it in your config you can block Athena from opening any applications until you are done with work for the day. https://github.com/friedmicro/athena

### Future improvements
- Caching for DB disconnections or crashes, currently a DB failure takes out even requesting previously set times
- Buffering\offloading times on the GTK client. If traveling with a laptop the time worked should sync back to the network once returned. Currently this requires a remote connection to the network which is mildly tedious.
- Improvements to the distraction daemon...the daemon currently is in C, but the pipewire integration isn't perfect. I will likely rewrite that daemon in the future to improve this.

### Project history
This project originally started as a monolithic flask application written in Python which fairly high performance requirements typically using around 100MB per replica. While this was a great prototype as time went on more and more of this project was rewritten into go. This is also why some API signatures show things like day numbers matching closer to python because that was the original design and it's why the DB has a small issue with timezones that requires a programmatic offset to correct today. It's been in use for a fairly long time and is something that I use daily at this point.

### Credits
Vee (https://github.com/PurposeUnknown): For her work on the work category system for work types.
