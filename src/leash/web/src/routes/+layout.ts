import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';
import advancedFormat from 'dayjs/plugin/advancedFormat';
import relativeTime from 'dayjs/plugin/relativeTime';
import { date_format } from '$lib/src/stores';
import Cookies from 'js-cookie';
import { DEFAULT_DATE_FORMAT } from '$lib/src/defaults';
import { getSelf, refreshTokens } from '$lib/src/leash';
import { user } from '$lib/src/stores';

export const prerender = true;
export const ssr = false;

dayjs.extend(utc);
dayjs.extend(timezone);
dayjs.extend(advancedFormat);
dayjs.extend(relativeTime);

if (!Cookies.get('date_format')) {
	Cookies.set('date_format', DEFAULT_DATE_FORMAT, { expires: 365, sameSite: 'strict' });
}

date_format.set(Cookies.get('date_format') || DEFAULT_DATE_FORMAT);
date_format.subscribe((value: string) => {
	Cookies.set('date_format', value, { expires: 365, sameSite: 'strict' });
});

refreshTokens().then((valid) => {
	if (valid) {
		getSelf().then((self) => {
			user.set(self);
		});
	} else {
		user.set(null);
	}
});