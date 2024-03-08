<script lang="ts">
	import { Calendar, type EventInput } from '@fullcalendar/core';
	import iCalendarPlugin from '@fullcalendar/icalendar';
	import dayGridPlugin from '@fullcalendar/daygrid';
	import timeGridPlugin from '@fullcalendar/timegrid';
	import listPlugin from '@fullcalendar/list';

	export let url: string;
	export let colorizeEvent: (event: EventInput) => void = () => {};

	function calendarAction(element: HTMLElement) {
		let calendar = new Calendar(element, {
			plugins: [dayGridPlugin, timeGridPlugin, listPlugin, iCalendarPlugin],
			events: {
				url,
				format: 'ics'
			},
			initialView: 'timeGridWeek',
			headerToolbar: {
				left: 'prev,next today',
				center: 'title',
				right: 'dayGridMonth,timeGridWeek,listWeek'
			},
			nowIndicator: true,
			scrollTime: new Date().getHours() + ':00:00',
			eventSourceSuccess: function (content, _) {
				for (const event of content) {
					if (event.color === undefined) {
						colorizeEvent(event);
					}
				}
			}
		});

		calendar.render();

		return {
			destroy: () => {
				calendar.destroy();
			}
		};
	}
</script>

<div
	class="flex aspect-video w-full justify-center divide-gray-100 border-gray-100 text-gray-700 dark:divide-gray-700 dark:border-gray-700 dark:text-gray-200"
>
	<div id="calendar" class="w-full" use:calendarAction />
</div>
