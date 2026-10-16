import { format, isToday, isTomorrow, addDays, setHours, setMinutes } from 'date-fns'
import { useI18n } from 'vue-i18n'

/**
 * Business hours composable providing generic business hours utilities.
 */
export function useBusinessHours () {
  const { t } = useI18n()

  function getBusinessHoursById (businessHoursId, businessHoursList) {
    if (!businessHoursId || !businessHoursList) {
      return null
    }

    return businessHoursList.find(bh => bh.id === businessHoursId)
  }

  function resolveBusinessHours (options) {
    const {
      showOfficeHours,
      showAfterAssignment,
      assignedBusinessHoursId,
      defaultBusinessHoursId,
      businessHoursList
    } = options

    if (!showOfficeHours) {
      return null
    }

    let businessHoursId = null

    // Check if we should use assigned business hours
    if (showAfterAssignment && assignedBusinessHoursId) {
      businessHoursId = assignedBusinessHoursId
    } else if (defaultBusinessHoursId) {
      // Fallback to default business hours
      businessHoursId = parseInt(defaultBusinessHoursId)
    }

    return getBusinessHoursById(businessHoursId, businessHoursList)
  }

  function isWithinBusinessHours (businessHours, date, utcOffset = 0) {
    if (!businessHours || businessHours.is_always_open) {
      return true
    }

    // Adjust for browser timezone: getTimezoneOffset() is negative for east of UTC,
    // which cancels out the browser offset that format/getDay will add.
    const adjustedOffset = utcOffset + date.getTimezoneOffset()
    const localDate = new Date(date.getTime() + (adjustedOffset * 60000))

    // Check if it's a holiday
    if (isHoliday(businessHours, localDate)) {
      return false
    }

    const dayName = getDayName(localDate.getDay())
    const ranges = getDayRanges(businessHours, dayName)

    if (!ranges || ranges.length === 0) {
      return false
    }

    const currentTime = format(localDate, 'HH:mm')
    return ranges.some(
      schedule =>
        schedule.open &&
        schedule.close &&
        schedule.open !== schedule.close &&
        currentTime >= schedule.open &&
        currentTime <= schedule.close
    )
  }

  // Returns the working time ranges of a day. Supports both the array format
  // {"Monday": [{"open", "close"}, ...]} and the legacy single object format
  // {"Monday": {"open", "close"}}.
  function getDayRanges (businessHours, dayName) {
    const schedule = businessHours?.hours?.[dayName]
    if (!schedule) {
      return []
    }
    return Array.isArray(schedule) ? schedule : [schedule]
  }

  function isHoliday (businessHours, date) {
    if (!businessHours.holidays || businessHours.holidays.length === 0) {
      return false
    }
    const dateStr = format(date, 'yyyy-MM-dd')
    return businessHours.holidays.some(holiday => {
      // Support date ranges (start_date/end_date) and the legacy single "date".
      const start = holiday.start_date || holiday.date
      const end = holiday.end_date || holiday.start_date || holiday.date
      if (!start) {
        return false
      }
      return dateStr >= start && dateStr <= end
    })
  }

  function getNextWorkingTime (businessHours, fromDate, utcOffset = 0) {
    if (!businessHours || businessHours.is_always_open) {
      return fromDate
    }

    // Check up to 14 days ahead
    for (let i = 0; i < 14; i++) {
      const checkDate = addDays(fromDate, i)
      const adjustedOffset = utcOffset + checkDate.getTimezoneOffset()
      const localDate = new Date(checkDate.getTime() + (adjustedOffset * 60000))

      // Skip holidays
      if (isHoliday(businessHours, localDate)) {
        continue
      }

      const dayName = getDayName(localDate.getDay())
      const ranges = getDayRanges(businessHours, dayName)
        .filter(schedule => schedule.open && schedule.close && schedule.open !== schedule.close)
        .sort((a, b) => a.open.localeCompare(b.open))

      if (ranges.length === 0) {
        continue
      }

      // Parse opening time
      const toLocalDate = (schedule, timeStr) => {
        const [hour, minute] = timeStr.split(':').map(Number)
        return setMinutes(setHours(localDate, hour), minute)
      }

      // Handle same-day logic
      if (i === 0) {
        const currentTime = format(localDate, 'HH:mm')
        for (const schedule of ranges) {
          // Past closing time of this range, try the next range of the day.
          if (currentTime >= schedule.close) {
            continue
          }
          // Currently within business hours.
          if (currentTime >= schedule.open) {
            return new Date(localDate.getTime() - (adjustedOffset * 60000))
          }
          // Before opening time of this range.
          return new Date(toLocalDate(schedule, schedule.open).getTime() - (adjustedOffset * 60000))
        }
        // Past all ranges today, check next day
        continue
      }

      // For future days, return the opening time of the first range
      const schedule = ranges[0]
      const nextWorking = toLocalDate(schedule, schedule.open)
      // Convert back from business timezone to user timezone
      return new Date(nextWorking.getTime() - (adjustedOffset * 60000))
    }

    return null
  }

  // Returns English day name to match backend hours keys
  function getDayName (dayNum) {
    return ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'][dayNum]
  }

  function formatNextWorkingTime (nextWorkingTime) {
    if (!nextWorkingTime) {
      return ''
    }

    if (isToday(nextWorkingTime)) {
      return t('globals.messages.backTodayAt', { time: format(nextWorkingTime, 'h:mm a') })
    } else if (isTomorrow(nextWorkingTime)) {
      return t('globals.messages.backTomorrowAt', { time: format(nextWorkingTime, 'h:mm a') })
    } else {
      return t('globals.messages.backOnDayAt', {
        day: format(nextWorkingTime, 'EEEE'),
        time: format(nextWorkingTime, 'h:mm a')
      })
    }
  }

  function getBusinessHoursStatus (businessHours, utcOffset = 0, withinHoursMessage = '') {
    if (!businessHours) {
      return null
    }

    const now = new Date()
    const within = isWithinBusinessHours(businessHours, now, utcOffset)

    let status = null
    if (within) {
      status = withinHoursMessage
    } else {
      const nextWorkingTime = getNextWorkingTime(businessHours, now, utcOffset)
      if (nextWorkingTime) {
        status = t('globals.messages.wellBeBack', { when: formatNextWorkingTime(nextWorkingTime) })
      } else {
        status = t('globals.messages.currentlyOffline')
      }
    }

    return { status, isWithin: within }
  }

  return {
    getBusinessHoursById,
    resolveBusinessHours,
    isWithinBusinessHours,
    getNextWorkingTime,
    formatNextWorkingTime,
    getBusinessHoursStatus,
    isHoliday,
    getDayName
  }
}
