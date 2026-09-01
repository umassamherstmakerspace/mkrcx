<script lang="ts">
	import { Calendar, type EventApi } from '@fullcalendar/core';
	import dayGridPlugin from '@fullcalendar/daygrid';
	import timeGridPlugin from '@fullcalendar/timegrid';
	import listPlugin from '@fullcalendar/list';
	import { Modal } from 'flowbite-svelte';
	import './calendar.pcss';

	export let url: string;
	export let detailsEnabled = false;
	export let fitWorkingHours = false;

	type EventDetails = {
		title: string;
		description: string;
		location: string;
		start: Date | null;
		end: Date | null;
		allDay: boolean;
	};

	let detailsOpen = false;
	let selectedEvent: EventDetails | null = null;

	function openEventDetails(event: EventApi) {
		if (!detailsEnabled) return;

		selectedEvent = {
			title: event.title || 'Untitled event',
			description:
				typeof event.extendedProps.description === 'string'
					? event.extendedProps.description.trim()
					: '',
			location:
				typeof event.extendedProps.location === 'string' ? event.extendedProps.location.trim() : '',
			start: event.start,
			end: event.end,
			allDay: event.allDay
		};
		detailsOpen = true;
	}

	function isSameDay(first: Date, second: Date) {
		return (
			first.getFullYear() === second.getFullYear() &&
			first.getMonth() === second.getMonth() &&
			first.getDate() === second.getDate()
		);
	}

	function formatEventTime(event: EventDetails) {
		if (!event.start) return 'Time not specified';

		const date = new Intl.DateTimeFormat(undefined, {
			weekday: 'long',
			month: 'long',
			day: 'numeric',
			year: 'numeric'
		});
		if (event.allDay) return `All day · ${date.format(event.start)}`;

		const time = new Intl.DateTimeFormat(undefined, {
			hour: 'numeric',
			minute: '2-digit'
		});
		if (!event.end) return `${date.format(event.start)} · ${time.format(event.start)}`;
		if (isSameDay(event.start, event.end)) {
			return `${date.format(event.start)} · ${time.format(event.start)}–${time.format(event.end)}`;
		}

		return `${date.format(event.start)} ${time.format(event.start)} – ${date.format(event.end)} ${time.format(event.end)}`;
	}

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
			scrollTime: fitWorkingHours ? '08:00:00' : new Date().getHours() + ':00:00',
			...(fitWorkingHours
				? {
						height: 'auto' as const,
						slotMinTime: '08:00:00',
						slotMaxTime: '22:00:00',
						slotDuration: '01:00:00'
					}
				: {}),
			eventClick: ({ event, jsEvent }) => {
				if (!detailsEnabled) return;
				jsEvent.preventDefault();
				openEventDetails(event);
			},
			eventDidMount: ({ el, event }) => {
				if (!detailsEnabled) return;

				el.tabIndex = 0;
				el.setAttribute('role', 'button');
				el.setAttribute('aria-label', `View details for ${event.title || 'untitled event'}`);
				el.onkeydown = (keyEvent) => {
					if (keyEvent.key !== 'Enter' && keyEvent.key !== ' ') return;
					keyEvent.preventDefault();
					openEventDetails(event);
				};
			},
			eventWillUnmount: ({ el }) => {
				el.onkeydown = null;
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
	class={`flex w-full justify-center divide-gray-100 border-gray-100 text-gray-700 dark:divide-gray-700 dark:border-gray-700 dark:text-gray-200 ${fitWorkingHours ? '' : 'md:aspect-video'}`}
>
	<div id="calendar" class:staff-calendar={detailsEnabled} class="w-full" use:calendarAction />
</div>

<Modal bind:open={detailsOpen} title={selectedEvent?.title || 'Event details'} size="md" autoclose>
	{#if selectedEvent}
		<div class="space-y-5 text-left text-gray-700 dark:text-gray-200">
			<p class="font-medium text-gray-900 dark:text-white">{formatEventTime(selectedEvent)}</p>

			{#if selectedEvent.location}
				<div>
					<h4
						class="text-sm font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400"
					>
						Location
					</h4>
					<p class="mt-1 whitespace-pre-wrap">{selectedEvent.location}</p>
				</div>
			{/if}

			<div>
				<h4 class="text-sm font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
					Details
				</h4>
				{#if selectedEvent.description}
					<p class="mt-1 whitespace-pre-wrap">{selectedEvent.description}</p>
				{:else}
					<p class="mt-1 text-gray-500 dark:text-gray-400">No additional details were provided.</p>
				{/if}
			</div>
		</div>
	{/if}
</Modal>
