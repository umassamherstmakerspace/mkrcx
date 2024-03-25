<script lang="ts">
	import { Calendar } from '@fullcalendar/core';
	import dayGridPlugin from '@fullcalendar/daygrid';
	import timeGridPlugin from '@fullcalendar/timegrid';
	import listPlugin from '@fullcalendar/list';
	import './calendar.pcss';

	export let url: string;

	function calendarAction(element: HTMLElement) {
		let calendar = new Calendar(element, {
			plugins: [dayGridPlugin, timeGridPlugin, listPlugin],
			events: {
				url
			},
			initialView: window.innerWidth < 768 ? 'listWeek' : 'timeGridWeek',
			headerToolbar: {
				left: 'prev,next today',
				center: 'title',
				right: 'dayGridMonth,timeGridWeek,listWeek'
			},
			nowIndicator: true,
			scrollTime: new Date().getHours() + ':00:00'
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
	class="flex w-full justify-center divide-gray-100 border-gray-100 text-gray-700 dark:divide-gray-700 dark:border-gray-700 dark:text-gray-200 md:aspect-video"
>
	<div id="calendar" class="w-full" use:calendarAction />
</div>
