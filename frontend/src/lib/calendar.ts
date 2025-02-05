import type { ICS } from '@filecage/ical';
import type { DateTime } from '@filecage/ical/ValueTypes';
import { parseString } from '@filecage/ical/parser';
import { addDays, isWithinInterval, subDays } from 'date-fns';
import RRULE from 'rrule';
import tcal from 'tcal';

export type EventJSON = {
	title: string;
	description?: string;
	start: string;
	end: string;
	allDay: boolean;
	uid: string;
	sequence: number;
	recurrenceId?: string;
};

export function getOffsetValue(offset: string): number {
	const regex = /([+-]?)(\d{2})(\d{2})(\d{2})?/;
	const match = offset.match(regex);

	if (!match) {
		throw new Error('Invalid offset');
	}

	const sign = match[1] === '-' ? -1 : 1;
	const hours = parseInt(match[2]);
	const minutes = parseInt(match[3]);
	const seconds = parseInt(match[4]) || 0;

	return sign * ((hours * 60 + minutes) * 60 + seconds) * 1000;
}

export class TimezoneTransition {
	public tzoffsetfrom: number;
	public tzoffsetto: number;
	public dtstart: Date;
	public rrule?: RRULE.RRuleSet;

	constructor(transition: ICS.TimezoneDefinition) {
		this.tzoffsetfrom = getOffsetValue(transition.TZOFFSETFROM.value);
		this.tzoffsetto = getOffsetValue(transition.TZOFFSETTO.value);
		this.dtstart = transition.DTSTART.value.date;
		if (transition.RRULE) {
			const rrule = new RRULE.RRuleSet();

			rrule.rrule(
				new RRULE.RRule({
					dtstart: this.dtstart,
					...RRULE.RRule.parseString(transition.RRULE.value)
				})
			);

			this.rrule = rrule;
		} else if (transition.RDATE) {
			throw new Error('RDATE not supported');
		}
	}

	public getLatestTransistion(date: Date): Date {
		if (this.rrule) {
			return this.rrule.before(date, true) || this.dtstart;
		} else {
			return this.dtstart;
		}
	}
}

export class Timezone {
	public tzid: string;
	public standardTransitions: TimezoneTransition[] = [];
	public daylightTransitions: TimezoneTransition[] = [];

	constructor(timezone: ICS.VTIMEZONE) {
		this.tzid = timezone.TZID.value;
		this.daylightTransitions = timezone.DAYLIGHT?.map((t) => new TimezoneTransition(t)) || [];
		this.standardTransitions = timezone.STANDARD?.map((t) => new TimezoneTransition(t)) || [];
	}

	public getUTCTime(date: Date): Date {
		const transitions = this.standardTransitions.concat(this.daylightTransitions).map((t) => {
			return { date: t.getLatestTransistion(date), offset: t.tzoffsetto };
		});

		transitions.sort((a, b) => b.date.getTime() - a.date.getTime());

		return new Date(date.getTime() - transitions[0].offset);
	}
}

function floatingTimestamp(date: Date): string {
	const str = date.toISOString();
	return str.substring(0, str.length - 1);
}

function getDateTimeTimestamp(date: DateTime, timezoneMap: TimezoneMap): string {
	if (date.isUTC) {
		return date.date.toISOString();
	} else {
		if (date.timezoneIdentifier === undefined) {
			return floatingTimestamp(date.date);
		} else {
			const tz = timezoneMap.get(date.timezoneIdentifier);
			if (!tz) {
				throw new Error(`Timezone ${date.timezoneIdentifier} not found`);
			}

			return tz.getUTCTime(date.date).toISOString();
		}
	}
}

function floatingTZAdjust(date: DateTime): Date {
	if (date.isUTC) {
		return date.date;
	}

	return new Date(date.date.getTime() - date.date.getTimezoneOffset() * 60 * 1000);
}

export type TimezoneMap = Map<string, Timezone>;

export type EventEndType =
	| {
			type: 'dateTime';
			difference: number;
	  }
	| {
			type: 'duration';
			difference: number;
	  };

export class EventInstance {
	public title: string;
	public description: string;
	public start: DateTime;
	public end: EventEndType;
	public allDay: boolean;
	public uid: string;
	public sequence: number;
	public recurrenceId?: DateTime;

	constructor(event: Event, start: Date) {
		this.title = event.title;
		this.description = event.description || '';
		this.start = {
			date: start,
			isDateOnly: event.start.isDateOnly,
			timezoneIdentifier: event.start.timezoneIdentifier,
			isUTC: event.start.isUTC
		} as DateTime;

		this.end = event.end;
		this.allDay = event.allDay;
		this.uid = event.uid;
		this.sequence = event.sequence;
		this.recurrenceId = event.recurrenceId;
	}

	public getEndTime(timezoneMap: TimezoneMap): DateTime {
		if (this.end.type === 'duration' && this.start.timezoneIdentifier) {
			const tz = timezoneMap.get(this.start.timezoneIdentifier);
			if (!tz) {
				throw new Error(`Timezone ${this.start.timezoneIdentifier} not found`);
			}

			const date = new Date(tz.getUTCTime(this.start.date).getTime() + this.end.difference);

			return {
				date,
				isDateOnly: this.start.isDateOnly,
				timezoneIdentifier: undefined,
				isUTC: true
			} as DateTime;
		} else {
			return {
				date: new Date(this.start.date.getTime() + this.end.difference),
				isDateOnly: this.start.isDateOnly,
				timezoneIdentifier: this.start.timezoneIdentifier,
				isUTC: this.start.isUTC
			} as DateTime;
		}
	}

	public getEndTimeTimestamp(timezoneMap: TimezoneMap): string {
		return getDateTimeTimestamp(this.getEndTime(timezoneMap), timezoneMap);
	}

	public getJSON(timezoneMap: TimezoneMap): EventJSON {
		const start = getDateTimeTimestamp(this.start, timezoneMap);
		return {
			title: this.title,
			description: this.description,
			start,
			end: this.getEndTimeTimestamp(timezoneMap),
			allDay: this.allDay,
			uid: this.uid,
			sequence: this.sequence,
			recurrenceId: this.recurrenceId
				? getDateTimeTimestamp(this.recurrenceId, timezoneMap)
				: undefined
		};
	}
}

export class Event {
	public title: string;
	public description?: string;
	public start: DateTime;
	public end: EventEndType;
	public allDay: boolean;
	public uid: string;
	public sequence: number;
	public recurrenceId?: DateTime;

	public rrules?: RRULE.RRuleSet;

	constructor(event: ICS.VEVENT.Published) {
		this.title = event.SUMMARY.value;
		this.description = event.DESCRIPTION?.value;

		this.start = event.DTSTART.value;
		if (!this.start.isUTC) {
			this.start = {
				date: floatingTZAdjust(this.start),
				isDateOnly: this.start.isDateOnly,
				timezoneIdentifier: this.start.timezoneIdentifier,
				isUTC: this.start.isUTC
			} as DateTime;
		}

		if (event.DTEND) {
			this.end = {
				type: 'dateTime',
				difference: event.DTEND.value.date.getTime() - event.DTSTART.value.date.getTime()
			};
		} else if (event.DURATION) {
			const dur = event.DURATION.value;
			let difference = dur.weeks || 0;
			difference *= 7;

			difference += dur.days || 0;
			difference *= 24;

			difference += dur.hours || 0;
			difference *= 60;

			difference += dur.minutes || 0;
			difference *= 60;

			difference += dur.seconds || 0;
			difference *= 1000;

			difference *= dur.inverted ? -1 : 1;

			this.end = {
				type: 'duration',
				difference
			};
		} else {
			throw new Error('Event must have either DTEND or DURATION');
		}

		this.allDay = event.DTSTART.value.isDateOnly;

		const rrules = new RRULE.RRuleSet();
		let useRrule = false;

		if (event.RRULE) {
			useRrule = true;
			rrules.rrule(
				new RRULE.RRule({
					dtstart: this.start.date,
					...RRULE.RRule.parseString(event.RRULE.value)
				})
			);
		}

		if (event.RDATE) {
			useRrule = true;
			throw new Error('RDATE not supported');
		}

		if (event.EXDATE && event.EXDATE.propertyList) {
			event.EXDATE.propertyList.forEach((exdates) => {
				exdates.value.forEach((exdate) => {
					rrules.exdate(floatingTZAdjust(exdate));
				});
			});
		}

		this.rrules = useRrule ? rrules : undefined;

		this.uid = event.UID.value;
		this.sequence = Number.parseInt(event.SEQUENCE?.value || '') || -1;
		if (event['RECURRENCE-ID']) {
			const id = event['RECURRENCE-ID'].value;
			this.recurrenceId = {
				date: floatingTZAdjust(id),
				isDateOnly: id.isDateOnly,
				timezoneIdentifier: id.timezoneIdentifier,
				isUTC: id.isUTC
			} as DateTime;
		}
	}

	public between(start: Date, end: Date): EventInstance[] {
		if (!this.rrules) {
			if (isWithinInterval(this.start.date, { start, end })) {
				return [new EventInstance(this, this.start.date)];
			} else {
				return [];
			}
		}

		const instances = this.rrules.between(
			subDays(new Date(start.getTime() - this.end.difference), 1),
			addDays(end.getTime(), 1),
			true
		);

		return instances.map((i) => new EventInstance(this, i));
	}
}

export class Calendar {
	public timezoneMap: TimezoneMap;
	public events: Event[];

	constructor(calendar: ICS.VCALENDAR) {
		this.timezoneMap = new Map();
		this.events = [];

		(calendar.VTIMEZONE || []).forEach((timezone) => {
			const tz = new Timezone(timezone);
			this.timezoneMap.set(timezone.TZID.value, tz);
		});

		this.events = (calendar.VEVENT || []).map((e) => new Event(e));
	}

	public between(start: Date, end: Date): EventJSON[] {
		const eventMap = new Map<string, EventJSON>();
		this.events.flatMap((e) => {
			const events = e.between(start, end).map((d) => d.getJSON(this.timezoneMap));

			events.forEach((event) => {
				const uid = `${event.uid}${event.recurrenceId || event.start}`;
				const existing = eventMap.get(uid);

				if (existing && !existing.recurrenceId && event.sequence >= existing.sequence) {
					eventMap.set(uid, event);
				} else {
					eventMap.set(uid, event);
				}
			});
		});

		return Array.from(eventMap.values());
	}
}

export class CalendarSet {
	public calendars: Calendar[];

	constructor(calendars: { VCALENDAR: ICS.VCALENDAR[] }) {
		this.calendars = calendars.VCALENDAR.map((c) => new Calendar(c));
	}

	public between(start: Date, end: Date): EventJSON[] {
		return this.calendars.flatMap((c) => c.between(start, end));
	}

	static cleanAndParse(input: string): CalendarSet {
		return new CalendarSet(parseString(cleanCalendar(input)));
	}
}

export function cleanCalendar(input: string): string {
	input = input.replace(/;FILENAME=/g, ';X-FILENAME='); // Fix non-standard FILENAME parameter from Google Calendar
	const data = tcal.parse(input);
	if (typeof data === 'string') {
		throw new Error('Invalid calendar');
	}

	data[2].forEach((component) => {
		if (component[0] === 'vevent') {
			if (component[1].find((property) => property[0] === 'summary') === undefined) {
				component[1].push(tcal.parse.property('SUMMARY:'));
			}
		}
	});

	return tcal.stringify(data);
}
