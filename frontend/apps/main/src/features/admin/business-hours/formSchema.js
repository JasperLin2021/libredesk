import * as z from 'zod'

const timeRegex = /^([01]\d|2[0-3]):([0-5]\d)$/

export const createFormSchema = (t) => z.object({
    name: z.string().min(1, t('globals.messages.required')),
    description: z.string().nullable().optional().transform(v => v ?? ''),
    is_always_open: z.boolean(),
    hours: z.record(
        z.array(
            z.object({
                open: z.string().regex(timeRegex, t('validation.invalidTimeFormat')),
                close: z.string().regex(timeRegex, t('validation.invalidTimeFormat')),
            })
        )
    ).optional()
}).superRefine((data, ctx) => {
    if (data.is_always_open === false) {
        if (!data.hours || Object.keys(data.hours).length === 0) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: t('globals.messages.required'),
                path: ['hours']
            })
        } else {
            for (const day in data.hours) {
                const ranges = data.hours[day]
                if (!Array.isArray(ranges) || ranges.length === 0) {
                    ctx.addIssue({
                        code: z.ZodIssueCode.custom,
                        message: t('globals.messages.required'),
                        path: ['hours', day]
                    })
                } else {
                    ranges.forEach((range, index) => {
                        if (!range?.open || !range?.close) {
                            ctx.addIssue({
                                code: z.ZodIssueCode.custom,
                                message: t('globals.messages.required'),
                                path: ['hours', day, index]
                            })
                        }
                    })
                }
            }
        }
    }
})
