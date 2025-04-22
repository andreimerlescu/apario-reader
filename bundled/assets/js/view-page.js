const triggerTabList = document.querySelectorAll('#myTab button')
triggerTabList.forEach(triggerEl => {
    const tabTrigger = new bootstrap.Tab(triggerEl)
    triggerEl.addEventListener('click', event => {
        event.preventDefault()
        tabTrigger.show()
    })
})
$(function(){
    function activate_lenszoom(){
        $('[data-toggle="lenszoom"]').imageZoom({
            zoomType: "lens",
            lensShape: "round",
            lensSize: 369,
            zoomLevel: 1.1776
        });
        $('[data-toggle="zoom"]').imageZoom();
    } //- /function

    activate_lenszoom();

    function deactivate_lenszoom(){
        let $new_page_contents = $("#page_contents").removeData().clone();
        $("#page_contents").remove();
        $(".zoomContainer").remove();
        $("#img-page-container").html($new_page_contents);
    } //- /function
});

Mousetrap.bind('right', function(){ window.location = "/page/{{ get_pg_id_from_doc_id_def_id_and_cur_pg_num .document_identifier .page_identifier ( plus .i_page 1 ) }}"}, 'keyup');
Mousetrap.bind('left', function (){ window.location = "/page/{{ get_pg_id_from_doc_id_def_id_and_cur_pg_num .document_identifier .page_identifier ( minus .i_page 1 ) }}"}, 'keyup')
Mousetrap.bind("f p enter", function() { window.location = "/page/{{ get_pg_id_from_doc_id_def_id_and_cur_pg_num .document_identifier .page_identifier 1 }}"}, "keyup")
Mousetrap.bind("l p enter", function (){ window.location = "/page/{{ get_pg_id_from_doc_id_def_id_and_cur_pg_num .document_identifier .page_identifier .i_total_pages }}"}, "keyup")
