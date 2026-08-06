# Calendar Sync

Your events are available as an iCal feed at:
```
http://your-server:8081/api/events/calendar.ics
```

## Google Calendar

1. Go to [calendar.google.com/calendar/r/settings/addbyurl](https://calendar.google.com/calendar/r/settings/addbyurl)
2. Paste the URL → **Add calendar**

> Syncs to the Google Calendar mobile app automatically. Google polls roughly every 24h — new events may not appear immediately.

## Apple Calendar (iPhone)

1. Settings → Calendar → Accounts → **Add Account**
2. **Other** → **Add Subscribed Calendar**
3. Paste the URL → Next → Save

## Apple Calendar (Mac)

1. File → **New Calendar Subscription**
2. Paste the URL → Subscribe

## Notes

- Read-only — changes made in the calendar app won't sync back to Zili
- If the app is down during a poll, the calendar keeps showing the last fetched data and retries next cycle
- To force a refresh in Google Calendar, remove and re-add the subscription

