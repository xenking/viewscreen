function footer() {
    $('.ui.checkbox').checkbox();
    $('.ui.dropdown').dropdown();

    $(document).on('click', '.confirm', function() {
        return confirm($(this).data('prompt'));
    });

    // Togglers
    $(document).on('click', '.toggler', function() {
        let target = $(this).siblings(".toggler");
        let action = $(this).data("action");
        let key = $(this).data("key");
        let value = $(this).data("value");

        let encoded = '';
        if (key && value) {
            encoded = key + "=" + encodeURIComponent(value)
        }

        $(this).toggle();
        $(target).toggle();
        $.ajax({ type: "POST", url: action, data: encoded });
    });

    // Set form input values.
    $('.set-input').click(function(e) {
        e.preventDefault();
        let target = $(this).data("target");
        let value = $(this).data("value");
        $(target).val(value);
    });

    // Close button.
    $('.message .close').on('click', function() {
        $(this).closest('.message').transition('fade');
        history.pushState(null, '', location.href.split('?')[0]);
    });

}
$(document).ready(footer);

