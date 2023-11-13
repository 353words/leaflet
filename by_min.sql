SELECT
    strftime('%H:%M', time),
    AVG(lat),
    AVG(lng)
FROM pts
    GROUP BY strftime('%H:%M', time)
;
