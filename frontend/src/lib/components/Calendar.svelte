<script lang="ts" context="module">
    interface Event {
        title: string;
        start: string;
        end: string;
    };
</script>

<script lang="ts">
    // @ts-nocheck
    import Calendar from '@event-calendar/core';
    import TimeGrid from '@event-calendar/time-grid';
    import DayGrid from '@event-calendar/day-grid';
    import List from '@event-calendar/list';
    import '@event-calendar/core/index.css';

    let plugins = [TimeGrid, DayGrid, List];
    let options = {
        view: 'timeGridWeek',
        headerToolbar: {
            start: 'prev,next today',
            center: 'title',
            end: 'dayGridMonth,timeGridWeek,timeGridDay,listWeek'
        },
        buttonText: function (texts) {
            texts.resourceTimeGridWeek = 'resources';
            return texts;
        },
        resources: [
            {id: 1, title: 'Resource A'},
            {id: 2, title: 'Resource B'}
        ],
        scrollTime: '09:00:00',
        // events: createEvents(),
        views: {
            timeGridWeek: {pointer: true},
            resourceTimeGridWeek: {pointer: true}
        },
        dayMaxEvents: true,
        nowIndicator: true,
        selectable: true
    };
</script>

<div class="w-full container flex justify-center bg-white dark:bg-gray-900 text-gray-700 dark:text-gray-200 border-gray-100 dark:border-gray-700 divide-gray-100 dark:divide-gray-700">
    <Calendar {plugins} {options} />
</div>

<style>
    :global(.ec) {
        flex: 1;
    }

    .container {
        aspect-ratio: 16/9;
    }
</style>