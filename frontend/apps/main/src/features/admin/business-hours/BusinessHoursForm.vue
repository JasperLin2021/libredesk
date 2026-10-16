<template>
  <form @submit="onSubmit" class="space-y-8">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>
          {{ t('globals.terms.name') }}
        </FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="description">
      <FormItem>
        <FormLabel>
          {{ t('globals.terms.description') }}
        </FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="is_always_open">
      <FormItem>
        <FormLabel>
          {{ t('admin.businessHours.setBusinessHours') }}
        </FormLabel>
        <FormControl>
          <RadioGroup v-bind="componentField">
            <div class="flex flex-col space-y-2">
              <div class="flex items-center space-x-3">
                <RadioGroupItem id="r1" :value="true" />
                <Label for="r1">{{ t('admin.businessHours.alwaysOpen24x7') }}</Label>
              </div>
              <div class="flex items-center space-x-3">
                <RadioGroupItem id="r2" :value="false" />
                <Label for="r2">{{ t('admin.businessHours.customBusinessHours') }}</Label>
              </div>
            </div>
          </RadioGroup>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="hours">
      <div v-if="form.values.is_always_open === false">
        <FormItem>
          <Tabs default-value="workdays">
            <TabsList class="grid w-full grid-cols-2">
              <TabsTrigger value="workdays">
                {{ t('businessHour.workdays') }}
              </TabsTrigger>
              <TabsTrigger value="holidays">
                {{ t('businessHour.holidays') }}
              </TabsTrigger>
            </TabsList>

            <!-- Workdays: multiple time ranges per day -->
            <TabsContent value="workdays" class="mt-4 space-y-3">
              <div
                v-for="day in WEEKDAYS"
                :key="day"
                class="flex items-start justify-between gap-4"
              >
                <div class="flex items-center space-x-3 pt-1 w-32 shrink-0">
                  <Checkbox
                    :id="day"
                    :checked="!!selectedDays[day]"
                    @update:checked="handleDayToggle(day, $event)"
                  />
                  <Label :for="day" class="font-medium">{{ t(`globals.weekdays.${day}`) }}</Label>
                </div>
                <div class="flex flex-col space-y-2 flex-1">
                  <div
                    v-for="(range, index) in hours[day] || []"
                    :key="index"
                    class="flex items-center justify-end space-x-2"
                  >
                    <Input
                      type="time"
                      class="w-32"
                      :modelValue="range.open"
                      @update:modelValue="(val) => updateHours(day, index, 'open', val)"
                      :disabled="!selectedDays[day]"
                    />
                    <span class="text-muted-foreground">-</span>
                    <Input
                      type="time"
                      class="w-32"
                      :modelValue="range.close"
                      @update:modelValue="(val) => updateHours(day, index, 'close', val)"
                      :disabled="!selectedDays[day]"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      :disabled="!selectedDays[day]"
                      :title="t('businessHour.addTimeRange')"
                      @click="addTimeRange(day)"
                    >
                      <Plus class="h-4 w-4" />
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      :disabled="!selectedDays[day] || (hours[day] || []).length <= 1"
                      :title="t('businessHour.removeTimeRange')"
                      @click="removeTimeRange(day, index)"
                    >
                      <Minus class="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>
            </TabsContent>

            <!-- Holidays: date ranges -->
            <TabsContent value="holidays" class="mt-4">
              <div class="flex justify-end items-center mb-4">
                <Button type="button" @click="startNewHoliday">
                  {{ t('businessHour.newHoliday') }}
                </Button>
              </div>
              <SimpleTable
                :headers="[
                  t('businessHour.holidayName'),
                  t('globals.terms.date')
                ]"
                :keys="['name', 'range']"
                :data="holidayRows"
                show-edit
                @editItem="startEditHoliday"
                @deleteItem="deleteHoliday"
              />
            </TabsContent>
          </Tabs>
          <FormMessage />
        </FormItem>
      </div>
    </FormField>

    <Dialog :open="openHolidayForm" @update:open="openHolidayForm = false">
      <DialogContent class="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {{
              isEditingHoliday ? t('businessHour.editHoliday') : t('businessHour.newHoliday')
            }}
          </DialogTitle>
          <DialogDescription />
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid grid-cols-4 items-center gap-4">
            <Label for="holiday_name" class="text-right"> {{ t('businessHour.holidayName') }} </Label>
            <Input id="holiday_name" ref="holidayNameRef" v-model="holidayName" class="col-span-3" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label for="start_date" class="text-right"> {{ t('businessHour.startDate') }} </Label>
            <Popover :open="startDatePickerOpen" @update:open="startDatePickerOpen = $event">
              <PopoverTrigger as-child>
                <Button
                  id="start_date"
                  variant="outline"
                  :class="
                    cn(
                      'col-span-3 justify-start text-left font-normal',
                      !holidayStartDate && 'text-muted-foreground'
                    )
                  "
                >
                  <CalendarIcon class="mr-2 h-4 w-4" />
                  {{ formatHolidayDate(holidayStartDate) || t('globals.terms.pickDate') }}
                </Button>
              </PopoverTrigger>
              <PopoverContent class="w-auto p-0">
                <Calendar v-model="holidayStartDate" @update:model-value="startDatePickerOpen = false" />
              </PopoverContent>
            </Popover>
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label for="end_date" class="text-right"> {{ t('businessHour.endDate') }} </Label>
            <Popover :open="endDatePickerOpen" @update:open="endDatePickerOpen = $event">
              <PopoverTrigger as-child>
                <Button
                  id="end_date"
                  variant="outline"
                  :class="
                    cn(
                      'col-span-3 justify-start text-left font-normal',
                      !holidayEndDate && 'text-muted-foreground'
                    )
                  "
                >
                  <CalendarIcon class="mr-2 h-4 w-4" />
                  {{ formatHolidayDate(holidayEndDate) || t('globals.terms.pickDate') }}
                </Button>
              </PopoverTrigger>
              <PopoverContent class="w-auto p-0">
                <Calendar v-model="holidayEndDate" @update:model-value="endDatePickerOpen = false" />
              </PopoverContent>
            </Popover>
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" @click="openHolidayForm = false">
            {{ t('globals.messages.cancel') }}
          </Button>
          <Button
            :disabled="
              !holidayName ||
              !holidayStartDate ||
              !holidayEndDate ||
              toDateStr(holidayEndDate) < toDateStr(holidayStartDate)
            "
            @click="saveHoliday"
          >
            {{ isEditingHoliday ? t('globals.messages.save') : t('globals.messages.add') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
    <Button type="submit" :disabled="isLoading" :isLoading="isLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import { ref, watch, reactive, computed, nextTick } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { RadioGroup, RadioGroupItem } from '@shared-ui/components/ui/radio-group/index.js'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@shared-ui/components/ui/form/index.js'
import { Calendar } from '@shared-ui/components/ui/calendar/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover/index.js'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs/index.js'
import { cn } from '@shared-ui/lib/utils.js'
import { format } from 'date-fns'
import { parseDate } from '@internationalized/date'
import { WEEKDAYS } from '../../../constants/date.js'
import { Calendar as CalendarIcon, Plus, Minus } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import SimpleTable from '@main/components/table/SimpleTable.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog/index.js'

const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: () => ''
  },
  isNewForm: {
    type: Boolean
  },
  isLoading: {
    type: Boolean,
    required: false
  }
})

const submitLabel = computed(() => {
  return props.submitLabel || (props.isNewForm ? t('globals.messages.create') : t('globals.messages.save'))
})

let holidays = reactive([])
const holidayName = ref('')
const holidayStartDate = ref(null)
const holidayEndDate = ref(null)
const editingHolidayIndex = ref(-1)
const selectedDays = ref({})
const hours = ref({})
const openHolidayForm = ref(false)
const startDatePickerOpen = ref(false)
const endDatePickerOpen = ref(false)
const holidayNameRef = ref(null)
const { t } = useI18n()

watch(openHolidayForm, (isOpen) => {
  if (isOpen) {
    nextTick(() => {
      holidayNameRef.value?.$el?.focus()
    })
  } else {
    // Reset the transient holiday form state when the dialog closes.
    editingHolidayIndex.value = -1
    holidayName.value = ''
    holidayStartDate.value = null
    holidayEndDate.value = null
  }
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
  initialValues: {
    description: '',
    is_always_open: true
  }
})

const isEditingHoliday = computed(() => editingHolidayIndex.value !== -1)

const holidayRows = computed(() =>
  holidays.map((h, index) => ({
    ...h,
    index,
    range: `${h.start_date} - ${h.end_date}`
  }))
)

// Serialize a calendar value (DateValue, Date or string) to 'yyyy-MM-dd'.
// DateValue.toString() already returns 'yyyy-MM-dd'; for JS dates the local
// calendar date is used so the stored value always matches the picked date
// regardless of the browser time zone.
const toDateStr = (value) => {
  if (!value) return null
  if (typeof value === 'string') return value
  if (value instanceof Date) return format(value, 'yyyy-MM-dd')
  return value.toString()
}

// Format a calendar value for display, e.g. 'MMMM dd, yyyy'.
const formatHolidayDate = (value) => {
  if (!value) return ''
  return format(new Date(`${toDateStr(value)}T00:00:00`), 'MMMM dd, yyyy')
}

const startNewHoliday = () => {
  editingHolidayIndex.value = -1
  holidayName.value = ''
  holidayStartDate.value = null
  holidayEndDate.value = null
  openHolidayForm.value = true
}

const startEditHoliday = (item) => {
  editingHolidayIndex.value = item?.index ?? -1
  holidayName.value = item?.name || ''
  // Calendar expects DateValue objects (from @internationalized/date),
  // passing plain JS Dates breaks the calendar.
  holidayStartDate.value = item?.start_date ? parseDate(item.start_date) : null
  holidayEndDate.value = item?.end_date ? parseDate(item.end_date) : null
  openHolidayForm.value = true
}

// Sync form field with local state
const syncHoursToForm = () => {
  form.setFieldValue('hours', { ...hours.value })
}

// Normalize hours to the array format, converting the legacy
// single object {"Monday": {"open", "close"}} to an array.
const normalizeHours = (rawHours) => {
  const result = {}
  Object.entries(rawHours || {}).forEach(([day, value]) => {
    result[day] = Array.isArray(value) ? value.map((v) => ({ ...v })) : [{ ...value }]
  })
  return result
}

// Normalize holidays to the date range format, converting the legacy
// {"name", "date"} to {"name", "start_date", "end_date"}.
const normalizeHolidays = (rawHolidays) =>
  (rawHolidays || []).map((h) => ({
    name: h.name,
    start_date: h.start_date || h.date,
    end_date: h.end_date || h.date
  }))

const saveHoliday = () => {
  const holiday = {
    name: holidayName.value,
    start_date: toDateStr(holidayStartDate.value),
    end_date: toDateStr(holidayEndDate.value)
  }

  if (isEditingHoliday.value && holidays[editingHolidayIndex.value]) {
    holidays[editingHolidayIndex.value] = holiday
  } else {
    holidays.push(holiday)
  }

  holidayName.value = ''
  holidayStartDate.value = null
  holidayEndDate.value = null
  editingHolidayIndex.value = -1
  openHolidayForm.value = false
}

const deleteHoliday = (item) => {
  const index = item?.index ?? -1
  if (index !== -1 && holidays[index]) {
    holidays.splice(index, 1)
  }
}

const handleDayToggle = (day, checked) => {
  selectedDays.value[day] = checked

  if (checked) {
    hours.value[day] = hours.value[day] || [{ open: '09:00', close: '17:00' }]
  } else {
    delete hours.value[day]
  }

  syncHoursToForm()
}

const addTimeRange = (day) => {
  if (!hours.value[day]) {
    hours.value[day] = [{ open: '09:00', close: '17:00' }]
  }
  const lastRange = hours.value[day][hours.value[day].length - 1]
  hours.value[day].push({ open: lastRange?.close || '09:00', close: lastRange?.close || '17:00' })
  syncHoursToForm()
}

const removeTimeRange = (day, index) => {
  if (!hours.value[day] || hours.value[day].length <= 1) {
    return
  }
  hours.value[day].splice(index, 1)
  syncHoursToForm()
}

const updateHours = (day, index, type, value) => {
  if (!hours.value[day]) {
    hours.value[day] = [{ open: '09:00', close: '17:00' }]
  }
  if (!hours.value[day][index]) {
    return
  }
  hours.value[day][index][type] = value
  syncHoursToForm()
}

const onSubmit = form.handleSubmit((values) => {
  const businessHours = values.is_always_open === true ? {} : { ...hours.value }

  const finalValues = {
    ...values,
    hours: businessHours,
    holidays: [...holidays]
  }
  props.submitForm(finalValues)
})

// Initialize state from props
const initializeFromValues = (values) => {
  if (!values) return

  // Reset state
  hours.value = {}
  selectedDays.value = {}
  holidays.length = 0
  editingHolidayIndex.value = -1

  // Set hours and selected days
  if (values.hours && typeof values.hours === 'object') {
    hours.value = normalizeHours(values.hours)
    selectedDays.value = Object.keys(hours.value).reduce((acc, day) => {
      acc[day] = true
      return acc
    }, {})
  }

  // Set holidays
  holidays.push(...normalizeHolidays(values.holidays))

  // Update form
  form.setValues(values, false)
  syncHoursToForm()
}

// Watch for initial values
watch(() => props.initialValues, initializeFromValues, { immediate: true, deep: true })
</script>
