// application.js

Mousetrap.bind('s i enter', function(){ window.location = "/StumbleInto"}, 'keyup');
Mousetrap.bind('q enter', function(){ window.location = "/StumbleInto"}, 'keyup');

$(function () {

    $('[data-role="tablist"] a').on('click', function (e) {
        e.preventDefault()
        $(this).tab('show')
    });

    $('.dropdown-toggle').dropdown();

    $('a[data-toggle="tab"]').on('shown.bs.tab', function (e) {
        var hash = $(e.target).attr('href');
        if (history.pushState) {
            history.pushState(null, null, hash);
        } else {
            location.hash = hash;
        }
    });

    var hash = window.location.hash;
    if (hash) {
        $('.nav-link[href="' + hash + '"]').tab('show');
    }
});